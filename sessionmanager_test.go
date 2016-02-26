package tobubus

import (
	"testing"
)

func TestGetUniqueSessionWithRecycleStrategy(t *testing.T) {
	manager := newSessionManager(recycleStrategy)

	if id := manager.getUniqueSessionID(); id != 0 {
		t.Errorf("expected 0, but %d", id)
	}
	if id := manager.getUniqueSessionID(); id != 1 {
		t.Errorf("expected 1, but %d", id)
	}
	if id := manager.getUniqueSessionID(); id != 2 {
		t.Errorf("expected 2, but %d", id)
	}
	go func() {
		channel := manager.getChannelOfSessionID(1)
		channel <- nil
	}()
	manager.receiveAndClose(1)
	if id := manager.getUniqueSessionID(); id != 1 {
		t.Errorf("expected 1, but %d", id)
	}
}

func TestGetUniqueSessionWithIncrementStrategy(t *testing.T) {
	manager := newSessionManager(incrementStrategy)

	if id := manager.getUniqueSessionID(); id != 0 {
		t.Errorf("expected 0, but %d", id)
	}
	if id := manager.getUniqueSessionID(); id != 1 {
		t.Errorf("expected 1, but %d", id)
	}
	if id := manager.getUniqueSessionID(); id != 2 {
		t.Errorf("expected 2, but %d", id)
	}
	go func() {
		channel := manager.getChannelOfSessionID(1)
		channel <- nil
	}()
	manager.receiveAndClose(1)
	if id := manager.getUniqueSessionID(); id != 3 {
		t.Errorf("expected 3, but %d", id)
	}
}
