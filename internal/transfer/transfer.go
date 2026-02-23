package transfer

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/pkg/sftp"
)

// TransferDirection indicates upload or download
type TransferDirection int

const (
	Download TransferDirection = iota
	Upload
)

// TransferStatus represents the state of a transfer item
type TransferStatus int

const (
	StatusPending TransferStatus = iota
	StatusInProgress
	StatusPaused
	StatusCompleted
	StatusFailed
	StatusCancelled
)

func (s TransferStatus) String() string {
	switch s {
	case StatusPending:
		return "Pending"
	case StatusInProgress:
		return "Transferring"
	case StatusPaused:
		return "Paused"
	case StatusCompleted:
		return "Completed"
	case StatusFailed:
		return "Failed"
	case StatusCancelled:
		return "Cancelled"
	}
	return "Unknown"
}

// TransferItem represents a single file transfer
type TransferItem struct {
	ID            int               `json:"id"`
	FileName      string            `json:"name"`
	RemotePath    string            `json:"remote_path"`
	LocalPath     string            `json:"local_path"`
	Direction     TransferDirection `json:"direction"`
	TotalBytes    int64             `json:"total_bytes"`
	TransferBytes int64             `json:"transfer_bytes"`
	Status        TransferStatus    `json:"status"`
	Error         error             `json:"-"`
	ErrorMsg      string            `json:"error"`
	StartTime     string            `json:"start_time"`
	EndTime       string            `json:"end_time"`
	cancel        chan struct{}     `json:"-"`
}

// Progress returns the transfer progress as a percentage (0-100)
func (t *TransferItem) Progress() int {
	if t.TotalBytes == 0 {
		return 0
	}
	return int(t.TransferBytes * 100 / t.TotalBytes)
}

// Speed returns current transfer speed in bytes per second
func (t *TransferItem) Speed(startTimeObj time.Time) int64 {
	if startTimeObj.IsZero() || t.TransferBytes == 0 {
		return 0
	}
	elapsed := time.Since(startTimeObj).Seconds()
	if elapsed == 0 {
		return 0
	}
	return int64(float64(t.TransferBytes) / elapsed)
}

// ETA returns estimated time remaining
func (t *TransferItem) ETA(startTimeObj time.Time) string {
	speed := t.Speed(startTimeObj)
	if speed == 0 {
		return "∞"
	}
	remaining := t.TotalBytes - t.TransferBytes
	seconds := remaining / speed
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	} else if seconds < 3600 {
		return fmt.Sprintf("%dm %ds", seconds/60, seconds%60)
	}
	return fmt.Sprintf("%dh %dm", seconds/3600, (seconds%3600)/60)
}

// FormatSize formats bytes into human-readable string
func FormatSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// FormatSpeed formats speed in bytes/sec to human readable
func FormatSpeed(bytesPerSec int64) string {
	return FormatSize(bytesPerSec) + "/s"
}

// TransferQueue manages a queue of file transfers
type TransferQueue struct {
	mu       sync.Mutex
	items    []*TransferItem
	nextID   int
	maxConc  int // max concurrent transfers
	active   int
	client   *sftp.Client
	onChange func() // callback when queue changes
}

// NewTransferQueue creates a new transfer queue
func NewTransferQueue(client *sftp.Client, maxConcurrent int) *TransferQueue {
	return &TransferQueue{
		items:   make([]*TransferItem, 0),
		maxConc: maxConcurrent,
		client:  client,
	}
}

// SetClient updates the SFTP client
func (q *TransferQueue) SetClient(client *sftp.Client) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.client = client
}

// SetOnChange sets a callback for when queue state changes
func (q *TransferQueue) SetOnChange(fn func()) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.onChange = fn
}

func (q *TransferQueue) notify() {
	if q.onChange != nil {
		q.onChange()
	}
}

// AddDownload adds a download to the queue
func (q *TransferQueue) AddDownload(remotePath, localPath string) *TransferItem {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.nextID++
	item := &TransferItem{
		ID:         q.nextID,
		FileName:   filepath.Base(remotePath),
		RemotePath: remotePath,
		LocalPath:  localPath,
		Direction:  Download,
		Status:     StatusPending,
		cancel:     make(chan struct{}),
	}

	q.items = append(q.items, item)
	go q.processNext()
	q.notify()
	return item
}

// AddUpload adds an upload to the queue
func (q *TransferQueue) AddUpload(localPath, remotePath string) *TransferItem {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.nextID++
	item := &TransferItem{
		ID:         q.nextID,
		FileName:   filepath.Base(localPath),
		RemotePath: remotePath,
		LocalPath:  localPath,
		Direction:  Upload,
		Status:     StatusPending,
		cancel:     make(chan struct{}),
	}

	q.items = append(q.items, item)
	go q.processNext()
	q.notify()
	return item
}

// CancelItem cancels a specific transfer
func (q *TransferQueue) CancelItem(id int) {
	q.mu.Lock()
	defer q.mu.Unlock()

	for _, item := range q.items {
		if item.ID == id {
			if item.Status == StatusInProgress {
				close(item.cancel)
			}
			item.Status = StatusCancelled
			break
		}
	}
	q.notify()
}

// ClearCompleted removes completed/failed/cancelled items
func (q *TransferQueue) ClearCompleted() {
	q.mu.Lock()
	defer q.mu.Unlock()

	var remaining []*TransferItem
	for _, item := range q.items {
		if item.Status == StatusPending || item.Status == StatusInProgress || item.Status == StatusPaused {
			remaining = append(remaining, item)
		}
	}
	q.items = remaining
	q.notify()
}

// GetItems returns a snapshot of all queue items
func (q *TransferQueue) GetItems() []*TransferItem {
	q.mu.Lock()
	defer q.mu.Unlock()

	result := make([]*TransferItem, len(q.items))
	copy(result, q.items)
	return result
}

// GetStats returns queue statistics
func (q *TransferQueue) GetStats() (pending, active, completed, failed int) {
	q.mu.Lock()
	defer q.mu.Unlock()

	for _, item := range q.items {
		switch item.Status {
		case StatusPending:
			pending++
		case StatusInProgress:
			active++
		case StatusCompleted:
			completed++
		case StatusFailed, StatusCancelled:
			failed++
		}
	}
	return
}

func (q *TransferQueue) processNext() {
	q.mu.Lock()
	if q.client == nil || q.active >= q.maxConc {
		q.mu.Unlock()
		return
	}

	// Find next pending item
	var nextItem *TransferItem
	for _, item := range q.items {
		if item.Status == StatusPending {
			nextItem = item
			break
		}
	}

	if nextItem == nil {
		q.mu.Unlock()
		return
	}

	startTimeObj := time.Now()
	nextItem.Status = StatusInProgress
	nextItem.StartTime = startTimeObj.Format(time.RFC3339)
	q.active++
	client := q.client
	q.mu.Unlock()

	q.notify()

	// Execute transfer
	var err error
	if nextItem.Direction == Download {
		err = q.doDownload(client, nextItem)
	} else {
		err = q.doUpload(client, nextItem)
	}

	q.mu.Lock()
	q.active--
	if err != nil {
		nextItem.Status = StatusFailed
		nextItem.Error = err
		nextItem.ErrorMsg = err.Error()
	} else if nextItem.Status == StatusInProgress {
		nextItem.Status = StatusCompleted
	}
	nextItem.EndTime = time.Now().Format(time.RFC3339)
	q.mu.Unlock()

	q.notify()

	// Process next item in queue
	q.processNext()
}

func (q *TransferQueue) doDownload(client *sftp.Client, item *TransferItem) (err error) {
	// Get remote file info for size
	info, err := client.Stat(item.RemotePath)
	if err != nil {
		return fmt.Errorf("stat remote: %w", err)
	}
	item.TotalBytes = info.Size()
	q.notify()

	// Open remote file
	src, err := client.Open(item.RemotePath)
	if err != nil {
		return fmt.Errorf("open remote: %w", err)
	}
	defer src.Close()

	// Create local file
	dst, err := os.Create(item.LocalPath)
	if err != nil {
		return fmt.Errorf("create local: %w", err)
	}
	defer dst.Close()

	// Cleanup on error
	defer func() {
		if err != nil {
			// Close file before removing
			dst.Close()
			os.Remove(item.LocalPath)
		}
	}()

	// Transfer with progress tracking
	buf := make([]byte, 32*1024)
	for {
		select {
		case <-item.cancel:
			return fmt.Errorf("cancelled")
		default:
		}

		n, readErr := src.Read(buf)
		if n > 0 {
			_, writeErr := dst.Write(buf[:n])
			if writeErr != nil {
				return fmt.Errorf("write: %w", writeErr)
			}
			item.TransferBytes += int64(n)
			q.notify()
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return fmt.Errorf("read: %w", readErr)
		}
	}
	return nil
}

func (q *TransferQueue) doUpload(client *sftp.Client, item *TransferItem) (err error) {
	// Get local file info
	info, err := os.Stat(item.LocalPath)
	if err != nil {
		return fmt.Errorf("stat local: %w", err)
	}
	item.TotalBytes = info.Size()
	q.notify()

	// Open local file
	src, err := os.Open(item.LocalPath)
	if err != nil {
		return fmt.Errorf("open local: %w", err)
	}
	defer src.Close()

	// Create remote file
	dst, err := client.Create(item.RemotePath)
	if err != nil {
		return fmt.Errorf("create remote: %w", err)
	}
	defer dst.Close()

	// Cleanup on error
	defer func() {
		if err != nil {
			dst.Close()
			client.Remove(item.RemotePath)
		}
	}()

	// Transfer with progress tracking
	buf := make([]byte, 32*1024)
	for {
		select {
		case <-item.cancel:
			return fmt.Errorf("cancelled")
		default:
		}

		n, readErr := src.Read(buf)
		if n > 0 {
			_, writeErr := dst.Write(buf[:n])
			if writeErr != nil {
				return fmt.Errorf("write: %w", writeErr)
			}
			item.TransferBytes += int64(n)
			q.notify()
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return fmt.Errorf("read: %w", readErr)
		}
	}
	return nil
}
