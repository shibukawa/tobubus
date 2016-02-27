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
		time.Sleep(2 * time.Millisecond)
		host.receiveMessage(socket)
		time.Sleep(time.Millisecond)
		wait <- "done"
	}()
	// receive Register from client here
	time.Sleep(time.Millisecond)
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
	time.Sleep(time.Millisecond)
	err := host.Unregister("github.com/shibukawa/tobubus/wrongname")
	if err == nil {
		t.Error("err should not be nil")
	}
	<-wait
	socket.Verify()
}

func TestHostUnregisterPluginFromPluginOK(t *testing.T) {
	host := newHostForTest("pipe.test")
	socket := mockconn.New(t)
	var pluginSessionID uint32 = 0
	socket.SetExpectedActions(
		// Receive Register from client here
		mockconn.Read(archiveMessage(RegisterClient, pluginSessionID, []byte("github.com/shibukawa/tobubus/1"))),
		mockconn.Write(archiveMessage(ResultOK, pluginSessionID, nil)),
		// Receive Unregister request from plugin
		mockconn.Read(archiveMessage(UnregisterClient, pluginSessionID+1, nil)),
		mockconn.Write(archiveMessage(ResultOK, pluginSessionID+1, nil)),
	)
	wait := make(chan string)
	go func() {
		host.receiveMessage(socket)
		time.Sleep(time.Millisecond)
		host.receiveMessage(socket)
		wait <- "done"
	}()
	// receive Register from client here
	// receive Unregister from client here
	<-wait
	if host.GetSocket("github.com/shibukawa/tobubus/1") != nil {
		t.Errorf("socket should be nil but %v", host.GetSocket("github.com/shibukawa/tobubus/1"))
	}
	socket.Verify()
}

func TestHostUnregisterPluginFromPluginNG(t *testing.T) {
	host := newHostForTest("pipe.test")
	socket := mockconn.New(t)
	wrongSocket := mockconn.New(t)
	var pluginSessionID uint32 = 0
	socket.SetExpectedActions(
		// Receive Register from client here
		mockconn.Read(archiveMessage(RegisterClient, pluginSessionID, []byte("github.com/shibukawa/tobubus/1"))),
		mockconn.Write(archiveMessage(ResultOK, pluginSessionID, nil)),
	)
	wrongSocket.SetExpectedActions(
		// Receive Unregister request from wrong plugin
		mockconn.Read(archiveMessage(UnregisterClient, pluginSessionID+1, nil)),
		mockconn.Write(archiveMessage(ResultNG, pluginSessionID+1, nil)),
	)
	wait := make(chan string)
	go func() {
		host.receiveMessage(socket)
		time.Sleep(time.Millisecond)
		host.receiveMessage(wrongSocket)
		wait <- "done"
	}()
	// receive Register from client here
	// receive Unregister from client here
	<-wait
	if host.GetSocket("github.com/shibukawa/tobubus/1") == nil {
		t.Error("socket should not be nil")
	}
	socket.Verify()
	wrongSocket.Verify()
}

func TestHostPublishAndCallFromHostOK(t *testing.T) {
	host := newHostForTest("pipe.test")
	obj := testStruct{result: "ok"}
	err := host.Publish("/image/reader", &obj)
	if err != nil {
		t.Errorf("error should be nil, but %v", err)
	}
	result, err := host.Call("/image/reader", "TestMethod", "test value")
	if err != nil {
		t.Errorf("error should be nil, but %v", err)
	}
	if len(result) != 1 {
		t.Errorf("obj.TestMethod should return one value, but %d result is returned", len(result))
	} else if result[0] != "ok" {
		t.Errorf("obj.TestMethod should return 'ok' but '%v' is returnd", result[0])
	}
	if len(obj.args) != 1 {
		t.Errorf("obj.TestMethod should be called with one argument, but %d argument is passed", len(obj.args))
	} else if obj.args[0] != "test value" {
		t.Errorf("obj.args[0] should be 'image.png', but %v", obj.args[0])
	}
}

func TestHostPublishAndCallFromHostNG(t *testing.T) {
	// Host -> Host
	host := newHostForTest("pipe.test")
	obj := testStruct{result: "ok"}
	err := host.Publish("/image/reader", &obj)
	if err != nil {
		t.Errorf("error should be nil, but %v", err)
	}
	_, err = host.Call("/image/reader/wrong", "TestMethod", "test value")
	if err == nil {
		t.Error("error should not be nil")
	}
}

func TestHostUnpublishOK(t *testing.T) {
	// Host -> Host
	host := newHostForTest("pipe.test")
	obj := testStruct{result: "ok"}
	err := host.Publish("/image/reader", &obj)
	if err != nil {
		t.Errorf("error should be nil, but %v", err)
	}
	err = host.Unpublish("/image/reader")
	if err != nil {
		t.Errorf("error should be nil, but %v", err)
	}
	_, err = host.Call("/image/reader", "TestMethod", "test value")
	if err == nil {
		t.Error("error should not be nil")
	}
}

func TestHostUnpublishNG(t *testing.T) {
	// Host -> Host
	host := newHostForTest("pipe.test")
	obj := testStruct{result: "ok"}
	err := host.Publish("/image/reader", &obj)
	if err != nil {
		t.Errorf("error should be nil, but %v", err)
	}
	err = host.Unpublish("/image/reader/wrong")
	if err == nil {
		t.Error("error should not be nil")
	}
}

func TestHostPublishAndCallFromPluginOK(t *testing.T) {
	// Host <- Plugin
	host := newHostForTest("pipe.test")
	obj := testStruct{result: "ok"}
	err := host.Publish("/image/reader", &obj)
	if err != nil {
		t.Errorf("error should be nil, but %v", err)
	}
	socket := mockconn.New(t)
	pluginSessionID := uint32(45)
	receive, _ := archiveMethodCallMessage(CallMethod, pluginSessionID, "/image/reader", "TestMethod", []interface{}{"image.png"})
	send, _ := archiveMethodCallMessage(ReturnMethod, pluginSessionID, "", "", []interface{}{"ok"})
	socket.SetExpectedActions(
		mockconn.Read(receive),
		mockconn.Write(send),
	)
	wait := make(chan string)
	go func() {
		host.receiveMessage(socket)
		time.Sleep(time.Millisecond)
		wait <- "done"
	}()
	// receive CallMethod from client here
	<-wait
	if len(obj.args) != 1 {
		t.Errorf("obj.TestMethod should be called with one argument, but %d argument is passed", len(obj.args))
	} else if obj.args[0] != "image.png" {
		t.Errorf("obj.args[0] should be 'image.png', but %v", obj.args[0])
	}
	socket.Verify()
}

func TestHostCallPluginFunctionOK(t *testing.T) {
	// Host -> Plugin
	host := newHostForTest("pipe.test")
	socket := mockconn.New(t)
	hostSessionID := host.sessions.getUniqueSessionID() + 1
	send, _ := archiveMethodCallMessage(CallMethod, hostSessionID, "/image/reader", "TestMethod", []interface{}{"test value"})
	receive, _ := archiveMethodCallMessage(ReturnMethod, hostSessionID, "", "", []interface{}{"ok"})
	pluginSessionID := uint32(1)
	socket.SetExpectedActions(
		mockconn.Read(archiveMessage(RegisterClient, pluginSessionID, []byte("github.com/shibukawa/tobubus/1"))),
		mockconn.Write(archiveMessage(ResultOK, pluginSessionID, nil)),
		mockconn.Read(archiveMessage(Publish, pluginSessionID+1, []byte("/image/reader"))),
		mockconn.Write(archiveMessage(ResultOK, pluginSessionID+1, nil)),
		mockconn.Write(send),
		mockconn.Read(receive),
	)
	wait := make(chan string)
	go func() {
		host.receiveMessage(socket)
		host.receiveMessage(socket)
		time.Sleep(2 * time.Millisecond)
		host.receiveMessage(socket)
		wait <- "done"
	}()
	time.Sleep(time.Millisecond)
	// receive Register from client here
	// receive Publish from client here
	result, err := host.Call("/image/reader", "TestMethod", "test value")
	<-wait
	if err != nil {
		t.Errorf("error should be nil, but %v", err)
	}
	if len(result) != 1 {
		t.Errorf("obj.TestMethod should return one argument, but %d result is returned", len(result))
	} else if result[0] != "ok" {
		t.Errorf("result[0] should be 'ok', but %v", result[0])
	}
	socket.Verify()
}

func TestHostLocalInstanceHasHigherPriority(t *testing.T) {
	// Host -> Plugin
	host := newHostForTest("pipe.test")
	obj := testStruct{result: "ok"}
	err := host.Publish("/image/reader", &obj)
	if err != nil {
		t.Errorf("error should be nil, but %v", err)
	}
	socket := mockconn.New(t)
	pluginSessionID := uint32(1)
	socket.SetExpectedActions(
		mockconn.Read(archiveMessage(RegisterClient, pluginSessionID, []byte("github.com/shibukawa/tobubus/1"))),
		mockconn.Write(archiveMessage(ResultOK, pluginSessionID, nil)),
		mockconn.Read(archiveMessage(Publish, pluginSessionID+1, []byte("/image/reader"))),
		mockconn.Write(archiveMessage(ResultOK, pluginSessionID+1, nil)),
	)
	wait := make(chan string)
	go func() {
		host.receiveMessage(socket)
		host.receiveMessage(socket)
		wait <- "done"
	}()
	time.Sleep(time.Millisecond)
	// receive Register from client here
	// receive Publish from client here
	_, err = host.Call("/image/reader", "TestMethod", "test value2")
	<-wait
	if err != nil {
		t.Errorf("error should be nil, but %v", err)
	}
	if len(obj.args) != 1 {
		t.Errorf("obj.TestMethod should be called with one argument, but %d argument is passed", len(obj.args))
	} else if obj.args[0] != "test value2" {
		t.Errorf("obj.args[0] should be 'test value2', but %v", obj.args[0])
	}
	socket.Verify()
}
