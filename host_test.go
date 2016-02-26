package tobubus

import (
	"github.com/shibukawa/mockconn"
	"testing"
	"time"
)

func TestHostRegisterAndUnregisterPluginOK(t *testing.T) {
	host := newHostForTest("pipe.test")
	socket := mockconn.New(t)
	var pluginSessionID uint32 = 0
	hostSessionID := host.sessions.getUniqueSessionID() + 1
	socket.SetExpectedActions(
		// Receive Register request
		mockconn.Read(archiveMessage(RegisterClient, pluginSessionID, []byte("github.com/shibukawa/tobubus/1"))),
		mockconn.Write(archiveMessage(ResultOK, pluginSessionID, nil)),
		// Send Unregister request from host
		mockconn.Write(archiveMessage(UnregisterClient, hostSessionID, nil)),
		mockconn.Read(archiveMessage(ResultOK, hostSessionID, nil)),
	)
	wait := make(chan string)
	go func() {
		host.receiveMessage(socket)
		time.Sleep(20 * time.Millisecond)
		host.receiveMessage(socket)
		time.Sleep(10 * time.Millisecond)
		wait <- "done"
	}()
	// receive Register from client here
	time.Sleep(10 * time.Millisecond)
	clientSocket := host.GetSocket("github.com/shibukawa/tobubus/1")
	if clientSocket == nil {
		t.Error("socket missing")
	} else if host.GetPluginID(clientSocket) != "github.com/shibukawa/tobubus/1" {
		t.Errorf("id is wrong: %s", host.GetPluginID(clientSocket))
	}
	host.Unregister("github.com/shibukawa/tobubus/1")
	<-wait
	socket.Verify()
}

func TestHostRegisterAndUnregisterPluginNG(t *testing.T) {
	host := newHostForTest("pipe.test")
	socket := mockconn.New(t)
	var pluginSessionID uint32 = 0
	socket.SetExpectedActions(
		// Receive Register request
		mockconn.Read(archiveMessage(RegisterClient, pluginSessionID, []byte("github.com/shibukawa/tobubus/1"))),
		mockconn.Write(archiveMessage(ResultOK, pluginSessionID, nil)),
	)
	wait := make(chan string)
	go func() {
		host.receiveMessage(socket)
		wait <- "done"
	}()
	// receive Register from client here
	time.Sleep(10 * time.Millisecond)
	err := host.Unregister("github.com/shibukawa/tobubus/wrongname")
	if err == nil {
		t.Error("err should not be nil")
	}
	<-wait
	socket.Verify()
}
