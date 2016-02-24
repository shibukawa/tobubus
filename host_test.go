package tobubus

/*import (
	"github.com/shibukawa/mockconn"
	"testing"
	"time"
)


func TestHostStart(t *testing.T) {
	host := NewHost("pipe.test")
	messageId := host.messageID() + 1
	socket.SetExpectedActions(
		mockconn.Write(archiveMessage(RegisterClient, messageId, []byte("github.com/shibukawa/tobubus/1"))),
		mockconn.Read(archiveMessage(ResultOK, messageId, nil)),
	)
	go func() {
		time.Sleep(10 * time.Millisecond)
		host.receiveMessage()
	}()
	host.Register()
	socket.Verify()
}*/
