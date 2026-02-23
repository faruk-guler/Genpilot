package transfer

import (
	"testing"
)

func TestQueueState(t *testing.T) {
	// Initialize with nil client to test queue logic only
	q := NewTransferQueue(nil, 2)

	if len(q.GetItems()) != 0 {
		t.Error("Queue should be empty")
	}

	// Add an item
	item := q.AddUpload("local", "remote")

	// Since client is nil, processNext should return early or fail gracefully without processing
	// In the current implementation, processNext checks `q.client == nil` and returns.
	// So the item should remain in StatusPending.
	if item.Status != StatusPending {
		t.Errorf("Expected pending, got %s", item.Status)
	}

	if len(q.GetItems()) != 1 {
		t.Errorf("Expected 1 item, got %d", len(q.GetItems()))
	}

	// Test cancellation
	q.CancelItem(item.ID)
	// CancelItem finds the item and sets status to Cancelled
	if item.Status != StatusCancelled {
		t.Errorf("Expected cancelled, got %s", item.Status)
	}
}
