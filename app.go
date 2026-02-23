package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"sync" // Import sync package for Mutex

	"Genpilot/internal/config"
	"Genpilot/internal/sftp"
	sshclient "Genpilot/internal/ssh"
	"Genpilot/internal/transfer" // Import transfer package

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"github.com/zalando/go-keyring"
	"golang.org/x/crypto/ssh"
)

// SessionState holds all resources for an active connection
type SessionState struct {
	ID            string
	Name          string
	SSHClient     *sshclient.Client
	SFTPClient    *sftp.Client
	SSHSession    *ssh.Session
	SSHStdin      io.WriteCloser
	TransferQueue *transfer.TransferQueue
	Tunnels       map[string]*sshclient.Tunnel
}

// App struct
type App struct {
	ctx          context.Context
	sessions     map[string]*SessionState
	sessionsLock sync.RWMutex
	sessionMgr   *config.SessionManager
}

// NewApp creates a new App application struct
func NewApp() *App {
	sm, _ := config.NewSessionManager()
	return &App{
		sessionMgr: sm,
		sessions:   make(map[string]*SessionState),
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// Connect establishes SSH connection and starts the shell for a specific session
func (a *App) Connect(id, name, host string, port int, user string, pass string) (string, error) {
	// Create SSH Client
	client, err := sshclient.NewClient(user, pass, host, port, 0, nil)
	if err != nil {
		return "", fmt.Errorf("auth failed: %w", err)
	}

	addr := fmt.Sprintf("%s:%d", host, port)
	err = client.Connect(addr)
	if err != nil {
		return "", fmt.Errorf("connection failed: %w", err)
	}

	// Create Session State
	state := &SessionState{
		ID:            id,
		Name:          name,
		SSHClient:     client,
		TransferQueue: transfer.NewTransferQueue(nil, 2),
		Tunnels:       make(map[string]*sshclient.Tunnel),
	}
	a.sessionsLock.Lock()
	a.sessions[id] = state
	a.sessionsLock.Unlock()

	// Set transfer queue callback
	state.TransferQueue.SetOnChange(func() {
		if a.ctx != nil {
			runtime.EventsEmit(a.ctx, "transfer-update-"+id, state.TransferQueue.GetItems())
		}
	})

	// Writer that emits events to frontend with ID
	writer := &eventWriter{
		ctx: a.ctx,
		id:  id,
		onLogout: func() {
			go func() {
				runtime.LogInfo(a.ctx, "Logout detected for "+id)
				a.DisconnectSession(id)
			}()
		},
	}

	// Prepare Shell
	session, err := client.PrepareShell(80, 24)
	if err != nil {
		return "", err
	}
	state.SSHSession = session

	// Setup Pipes
	stdoutPipe, err := session.StdoutPipe()
	if err != nil {
		return "", err
	}
	stderrPipe, err := session.StderrPipe()
	if err != nil {
		return "", err
	}
	stdinPipe, err := session.StdinPipe()
	if err != nil {
		return "", err
	}
	state.SSHStdin = stdinPipe

	// Start Shell
	if err := session.Shell(); err != nil {
		return "", fmt.Errorf("shell failed: %w", err)
	}

	// Start Copyroutines
	go func() {
		_, _ = io.Copy(writer, stdoutPipe)
		if state.SSHStdin != nil {
			state.SSHStdin.Close()
		}
		if session != nil {
			_ = session.Close()
		}
	}()

	go func() {
		_, _ = io.Copy(writer, stderrPipe)
	}()

	// Monitor session closure
	go func() {
		_ = session.Wait()
		if a.ctx != nil {
			runtime.EventsEmit(a.ctx, "disconnected-"+id, "Disconnected")
		}
		a.DisconnectSession(id)
	}()

	// Initialize SFTP
	sftpClient, err := sftp.NewClient(client.GetClient())
	if err == nil {
		state.SFTPClient = sftpClient
		state.TransferQueue.SetClient(sftpClient.GetSFTPClient())
	}

	return "Connected", nil
}

func (a *App) ResizeTerminal(id string, rows, cols int) {
	a.sessionsLock.RLock()
	s, ok := a.sessions[id]
	a.sessionsLock.RUnlock()

	if ok && s.SSHClient != nil && s.SSHSession != nil {
		s.SSHClient.ResizeTerminal(s.SSHSession, rows, cols)
	}
}

func (a *App) WriteToTerminal(id string, data string) {
	a.sessionsLock.RLock()
	s, ok := a.sessions[id]
	a.sessionsLock.RUnlock()

	if ok && s.SSHStdin != nil {
		s.SSHStdin.Write([]byte(data))
	}
}

// eventWriter implements io.Writer and emits events
type eventWriter struct {
	ctx      context.Context
	id       string
	onLogout func()
}

func (w *eventWriter) Write(p []byte) (n int, err error) {
	s := string(p)
	lower := strings.ToLower(s)
	if strings.Contains(lower, "logout") || strings.Contains(lower, "connection closed") {
		if w.onLogout != nil {
			w.onLogout()
		}
	}
	runtime.EventsEmit(w.ctx, "terminal-data-"+w.id, s)
	return len(p), nil
}

// SFTP Methods
type FileItem struct {
	Name  string `json:"name"`
	Size  string `json:"size"`
	Mode  string `json:"mode"`
	Time  string `json:"time"`
	IsDir bool   `json:"is_dir"`
}

func (a *App) ListFiles(id, path string) ([]FileItem, error) {
	a.sessionsLock.RLock()
	s, ok := a.sessions[id]
	a.sessionsLock.RUnlock()

	if !ok || s.SFTPClient == nil {
		return nil, fmt.Errorf("SFTP not connected for session %s", id)
	}
	if path == "" {
		path = "."
	}

	entries, err := s.SFTPClient.ListDirectory(path)
	if err != nil {
		return nil, err
	}

	var files []FileItem
	for _, e := range entries {
		files = append(files, FileItem{
			Name:  e.Name(),
			Size:  fmt.Sprintf("%d", e.Size()),
			Mode:  e.Mode().String(),
			Time:  e.ModTime().Format("2006-01-02 15:04"),
			IsDir: e.IsDir(),
		})
	}
	return files, nil
}

func (a *App) GoUp(id, path string) string {
	a.sessionsLock.RLock()
	s, ok := a.sessions[id]
	a.sessionsLock.RUnlock()

	if ok && s.SFTPClient != nil {
		return s.SFTPClient.Join(path, "..")
	}
	return path
}

// Session Management Methods

func (a *App) SaveSession(name, host, user, pass, group string, port int) error {
	session := config.Session{
		Name:     name,
		Host:     host,
		Port:     port,
		Username: user,
		Password: pass,
		Group:    group,
	}
	return a.sessionMgr.AddSession(session)
}

func (a *App) GetSessionPassword(name string) string {
	pass, err := keyring.Get("Genpilot", name)
	if err != nil {
		return ""
	}
	return pass
}

func (a *App) LoadSessions() []config.Session {
	return a.sessionMgr.GetAllSessions()
}

func (a *App) DeleteSession(name string) error {
	return a.sessionMgr.DeleteSession(name)
}

func (a *App) DisconnectSession(id string) {
	a.sessionsLock.Lock()
	s, ok := a.sessions[id]
	if !ok {
		a.sessionsLock.Unlock() // Release if not found
		return
	}
	// We found it, now we also remove it from the map immediately so no one else picks it up
	delete(a.sessions, id)
	a.sessionsLock.Unlock()

	runtime.LogInfo(a.ctx, "Disconnecting session "+id)

	for _, tunnel := range s.Tunnels {
		tunnel.Stop()
	}

	if s.SSHStdin != nil {
		s.SSHStdin.Close()
	}
	if s.SSHSession != nil {
		s.SSHSession.Close()
	}
	if s.SSHClient != nil {
		s.SSHClient.Close()
	}

	// removed from sessions map above

	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "disconnected-"+id, "Disconnected")
	}
}

func (a *App) DisconnectAll() {
	a.sessionsLock.RLock()
	// Create a slice of IDs to disconnect to avoid locking issues during iteration
	ids := make([]string, 0, len(a.sessions))
	for id := range a.sessions {
		ids = append(ids, id)
	}
	a.sessionsLock.RUnlock()

	for _, id := range ids {
		a.DisconnectSession(id)
	}
}

// File Transfer Methods - UPDATED to use Queue

func (a *App) DownloadFile(id, remotePath string, localPath string) error {
	a.sessionsLock.RLock()
	s, ok := a.sessions[id]
	a.sessionsLock.RUnlock()

	if !ok || s.SFTPClient == nil {
		return fmt.Errorf("not connected for session %s", id)
	}
	if s.TransferQueue != nil {
		s.TransferQueue.AddDownload(remotePath, localPath)
	}
	return nil
}

// UploadFile uploads a file from local to remote
func (a *App) UploadFile(id, localPath string, remotePath string) error {
	a.sessionsLock.RLock()
	s, ok := a.sessions[id]
	a.sessionsLock.RUnlock()

	if !ok || s.SFTPClient == nil {
		return fmt.Errorf("not connected for session %s", id)
	}
	if s.TransferQueue != nil {
		s.TransferQueue.AddUpload(localPath, remotePath)
	}
	return nil
}

// RenameFile renames a remote file
func (a *App) RenameFile(id, oldPath, newPath string) error {
	a.sessionsLock.RLock()
	s, ok := a.sessions[id]
	a.sessionsLock.RUnlock()

	if ok && s.SFTPClient != nil {
		return s.SFTPClient.Rename(oldPath, newPath)
	}
	return fmt.Errorf("not connected for session %s", id)
}

// DeleteRemoteFile deletes a remote file or directory
func (a *App) DeleteRemoteFile(id, path string) error {
	a.sessionsLock.RLock()
	s, ok := a.sessions[id]
	a.sessionsLock.RUnlock()

	if !ok || s.SFTPClient == nil {
		return fmt.Errorf("not connected for session %s", id)
	}
	stat, err := s.SFTPClient.Stat(path)
	if err != nil {
		return err
	}
	if stat.IsDir() {
		return s.SFTPClient.RemoveDirectory(path)
	}
	return s.SFTPClient.Remove(path)
}

// GetTransfers returns current transfer items
func (a *App) GetTransfers(id string) []*transfer.TransferItem {
	a.sessionsLock.RLock()
	s, ok := a.sessions[id]
	a.sessionsLock.RUnlock()

	if ok && s.TransferQueue != nil {
		return s.TransferQueue.GetItems()
	}
	return nil
}

// CancelTransfer cancels a transfer by ID
func (a *App) CancelTransfer(id string, transferID int) {
	a.sessionsLock.RLock()
	s, ok := a.sessions[id]
	a.sessionsLock.RUnlock()

	if ok && s.TransferQueue != nil {
		s.TransferQueue.CancelItem(transferID)
	}
}

// ClearCompletedTransfers clears finished items
func (a *App) ClearCompletedTransfers(id string) {
	a.sessionsLock.RLock()
	s, ok := a.sessions[id]
	a.sessionsLock.RUnlock()

	if ok && s.TransferQueue != nil {
		s.TransferQueue.ClearCompleted()
	}
}

// UI Helpers

func (a *App) SelectUploadFile() (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select File to Upload",
	})
}

func (a *App) SelectSavePath(filename string) (string, error) {
	return runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Save File As",
		DefaultFilename: filename,
	})
}

// Local Filesystem Methods

func (a *App) ListLocalFiles(path string) ([]FileItem, error) {
	if path == "" {
		var err error
		path, err = os.Getwd()
		if err != nil {
			return nil, err
		}
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	var files []FileItem
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}
		files = append(files, FileItem{
			Name:  e.Name(),
			Size:  fmt.Sprintf("%d", info.Size()),
			Mode:  info.Mode().String(),
			Time:  info.ModTime().Format("2006-01-02 15:04"),
			IsDir: e.IsDir(),
		})
	}
	return files, nil
}

func (a *App) GetLocalWD() (string, error) {
	return os.Getwd()
}

// TunnelInfo is used to send tunnel data to the frontend
type TunnelInfo struct {
	ID         string `json:"id"`
	LocalPort  int    `json:"local_port"`
	RemoteHost string `json:"remote_host"`
	RemotePort int    `json:"remote_port"`
}

// StartLocalForward initiates a new SSH local port forwarding tunnel
func (a *App) StartLocalForward(id string, localPort int, remoteHost string, remotePort int) error {
	a.sessionsLock.RLock()
	s, ok := a.sessions[id]
	a.sessionsLock.RUnlock()

	if !ok || s.SSHClient == nil {
		return fmt.Errorf("session %s not connected", id)
	}

	tunnelId := fmt.Sprintf("%d:%s:%d", localPort, remoteHost, remotePort)

	a.sessionsLock.Lock()
	defer a.sessionsLock.Unlock()

	if _, exists := s.Tunnels[tunnelId]; exists {
		return fmt.Errorf("tunnel on local port %d already exists", localPort)
	}

	tunnel, err := s.SSHClient.StartLocalForward(tunnelId, localPort, remoteHost, remotePort)
	if err != nil {
		return err
	}

	s.Tunnels[tunnelId] = tunnel
	return nil
}

// StopLocalForward stops an active SSH local port forwarding tunnel
func (a *App) StopLocalForward(id string, tunnelId string) error {
	a.sessionsLock.Lock()
	defer a.sessionsLock.Unlock()

	s, ok := a.sessions[id]
	if !ok {
		return fmt.Errorf("session %s not found", id)
	}

	tunnel, exists := s.Tunnels[tunnelId]
	if !exists {
		return fmt.Errorf("tunnel %s not found", tunnelId)
	}

	tunnel.Stop()
	delete(s.Tunnels, tunnelId)
	return nil
}

// GetActiveTunnels returns a list of active tunnels for the session
func (a *App) GetActiveTunnels(id string) []TunnelInfo {
	a.sessionsLock.RLock()
	defer a.sessionsLock.RUnlock()

	var result []TunnelInfo
	s, ok := a.sessions[id]
	if !ok {
		return result
	}

	for _, t := range s.Tunnels {
		result = append(result, TunnelInfo{
			ID:         t.ID,
			LocalPort:  t.LocalPort,
			RemoteHost: t.RemoteHost,
			RemotePort: t.RemotePort,
		})
	}
	return result
}
