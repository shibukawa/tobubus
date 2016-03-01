package tobubus

import (
	"bytes"
	"github.com/shibukawa/mockconn"
	"testing"
)

func TestArchiveMessage(t *testing.T) {
	data := archiveMessage(ConnectClient, 0, []byte("github.com/shibukawa/tobubus/1"))
	expected := []byte("\x10\x00\x00\x00\x00\x00\x00\x00\x1e\x00\x00\x00github.com/shibukawa/tobubus/1")
	if bytes.Compare(expected, data) != 0 {
		t.Errorf("archive error. expected='%v' actual='%v'", expected, data)
	}
}

func TestParseMessage(t *testing.T) {
	socket := mockconn.New(t)
	socket.SetExpectedActions(
		mockconn.Read([]byte("\x10\x00\x00\x00\x00\x00\x00\x00\x1e\x00\x00\x00github.com/shibukawa/tobubus/1")),
	)
	message, err := parseMessage(socket)
	if err != nil {
		t.Errorf("err should be nil, but %v", err)
	}
	if message.Type != ConnectClient {
		t.Errorf("parse error: %v", message.Type)
	}
	if string(message.body) != "github.com/shibukawa/tobubus/1" {
		t.Errorf("parse error: %s", string(message.body))
	}
}

func TestArchiveMethodCall(t *testing.T) {
	data, err := archiveMethodCallMessage(ConnectClient, 0, "/image", "open", []interface{}{"test.png", 0777})
	if err != nil {
		t.Errorf("err should be nil, but %v", err)
		return
	}
	socket := mockconn.New(t)
	socket.SetExpectedActions(
		mockconn.Read(data),
	)
	return
	message, err := parseMessage(socket)
	if string(message.body) != `{path:"image",method:"open",param:["test.png",511]}` {
		t.Errorf("archive error: %v", string(message.body))
	}
}
