package tobubus

import (
	"encoding/binary"
	"encoding/json"
	"io"
	"net"
)

type MessageType uint32

const (
	ResultOK             MessageType = 0x1
	ResultNG                         = 0x2
	ResultObjectNotFound             = 0x3
	ResultMethodNotFound             = 0x4
	RegisterClient                   = 0x10
	UnregisterClient                 = 0x11
	ConfirmPath                      = 0x20
	Publish                          = 0x21
	Unpublish                        = 0x22
	CallMethod                       = 0x30
	ReturnMethod                     = 0x31
)

type message struct {
	Type MessageType
	ID   uint32
	body []byte
}

type methodCall struct {
	Path   string        `json:"path,omitempty"`
	Method string        `json:"method,omitempty"`
	Params []interface{} `json:"params"`
}

func archiveMessage(msg MessageType, sessionID uint32, body []byte) []byte {
	result := make([]byte, 12+len(body))
	binary.LittleEndian.PutUint32(result, uint32(msg))
	binary.LittleEndian.PutUint32(result[4:], sessionID)
	binary.LittleEndian.PutUint32(result[8:], uint32(len(body)))
	copy(result[12:], body)
	return result
}

func parseMessage(conn net.Conn) (*message, error) {
	header := make([]byte, 12)
	_, err := io.ReadAtLeast(conn, header, 12)
	if err != nil {
		return nil, err
	}
	messageType := binary.LittleEndian.Uint32(header)
	sessionID := binary.LittleEndian.Uint32(header[4:])
	bodySize := binary.LittleEndian.Uint32(header[8:])
	var body []byte
	if bodySize > 0 {
		body = make([]byte, bodySize)
		_, err = io.ReadAtLeast(conn, body, int(bodySize))
		if err != nil {
			return nil, err
		}
	}
	return &message{
		Type: MessageType(messageType),
		ID:   sessionID,
		body: body,
	}, nil
}

func archiveMethodCallMessage(msg MessageType, msgID uint32, path, methodName string, params []interface{}) ([]byte, error) {
	src := methodCall{
		Path:   path,
		Method: methodName,
		Params: params,
	}
	data, err := json.Marshal(src)
	if err != nil {
		return nil, err
	}
	return archiveMessage(msg, msgID, data), nil
}
