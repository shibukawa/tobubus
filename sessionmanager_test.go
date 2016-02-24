package tobubus

import (
	"testing"
)

func TestGetUniqueSession(t *testing.T) {
	manager := newSessionManager()

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
