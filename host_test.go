package tobubus

import (
	"github.com/shibukawa/mockconn"
	"testing"
	"time"
)

func TestHostStartAndRegisterAndUnregisterPlugin(t *testing.T) {
	host := NewHost("pipe.test")
	socket := mockconn.New(t)
	sessionID := host.sessions.getUniqueSessionID() + 1
	socket.SetExpectedActions(
		mockconn.Read(archiveMessage(RegisterClient, sessionID, []byte("github.com/shibukawa/tobubus/1"))),
		mockconn.Write(archiveMessage(ResultOK, sessionID, nil)),
		mockconn.Write(archiveMessage(UnregisterClient, sessionID, nil)),
		mockconn.Read(archiveMessage(ResultOK, sessionID, nil)),
	)
	wait := make(chan string)
	go func() {
		host.receiveMessage(socket)
		time.Sleep(20 * time.Millisecond)
		host.receiveMessage(socket)
		time.Sleep(10 * time.Millisecond)
		wait <- "done"
	}()
	time.Sleep(10 * time.Millisecond)
	host.Unregister("github.com/shibukawa/tobubus/1")
	<-wait
	socket.Verify()
}
