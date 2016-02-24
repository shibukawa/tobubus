package tobubus

import (
	"encoding/json"
	"fmt"
	"github.com/shibukawa/localsocket"
	"net"
)

type Plugin struct {
	pipeName string
	id       string
	socket   net.Conn
	sessions *sessionManager

	objectMap map[string]*Proxy
}

func NewPlugin(pipeName, id string) (*Plugin, error) {
	socket, err := localsocket.NewLocalSocket(pipeName)
	if err != nil {
		return nil, err
	}
	return &Plugin{
		pipeName:  pipeName,
		id:        id,
		socket:    socket,
		objectMap: make(map[string]*Proxy),
		sessions:  newSessionManager(),
	}, nil
}

func (p *Plugin) receiveMessage() error {
	msg, err := parseMessage(p.socket)
	if err != nil {
		return err
	}
	switch msg.Type {
	case ResultOK, ResultNG, ReturnMethod:
		channel := p.sessions.getChannelOfSessionID(msg.ID)
		channel <- msg
	case CallMethod:
		go func() {
			method := &methodCall{}
			json.Unmarshal(msg.body, method)
			obj, ok := p.objectMap[method.Path]
			if !ok {
				p.socket.Write(archiveMessage(ResultObjectNotFound, msg.ID, nil))
				return
			}
			result, err := obj.Call(method.Method, method.Params...)
			if err != nil {
				p.socket.Write(archiveMessage(ResultMethodNotFound, msg.ID, nil))
			} else {
				resultMessage, err := archiveMethodCallMessage(ReturnMethod, msg.ID, "", "", result)
				if err != nil {
					p.socket.Write(archiveMessage(ResultNG, msg.ID, nil))
				}
				p.socket.Write(resultMessage)
			}
		}()
	case UnregisterClient, ConfirmPath:
		// todo
	case RegisterClient:
		p.socket.Write(archiveMessage(ResultNG, msg.ID, nil))
	}
	return nil
}

func (p *Plugin) Register() error {
	sessionID := p.sessions.getUniqueSessionID()
	p.socket.Write(archiveMessage(RegisterClient, sessionID, []byte(p.id)))
	message := p.sessions.receiveAndClose(sessionID)
	if message.Type != ResultOK {
		p.socket.Close()
		return fmt.Errorf("Can't connect to '%s'", p.pipeName)
	}
	return nil
}

func (p *Plugin) Unregister() error {
	sessionID := p.sessions.getUniqueSessionID()
	p.socket.Write(archiveMessage(UnregisterClient, sessionID, nil))
	message := p.sessions.receiveAndClose(sessionID)
	p.socket.Close()
	p.socket = nil
	if message.Type != ResultOK {
		return fmt.Errorf("Unregister error: '%s'", p.pipeName)
	}
	return nil
}

func (p *Plugin) ConfirmPath(path string) bool {
	sessionID := p.sessions.getUniqueSessionID()
	p.socket.Write(archiveMessage(ConfirmPath, sessionID, []byte(path)))
	message := p.sessions.receiveAndClose(sessionID)
	return message.Type == ResultOK
}

func (p *Plugin) Publish(path string, service interface{}) error {
	proxy, err := NewProxy(service)
	if _, ok := p.objectMap[path]; ok {
		p.objectMap[path] = proxy
		return nil
	}
	if err != nil {
		return err
	}
	sessionID := p.sessions.getUniqueSessionID()
	p.socket.Write(archiveMessage(Publish, sessionID, []byte(path)))
	message := p.sessions.receiveAndClose(sessionID)
	if message.Type != ResultOK {
		return fmt.Errorf("Can't publish object at '%s'", path)
	}
	p.objectMap[path] = proxy
	return nil
}

func (p *Plugin) Unpublish(path string) error {
	if _, ok := p.objectMap[path]; !ok {
		return fmt.Errorf("No object is published at '%s'", path)
	}
	sessionID := p.sessions.getUniqueSessionID()
	p.socket.Write(archiveMessage(Unpublish, sessionID, []byte(path)))
	message := p.sessions.receiveAndClose(sessionID)
	if message.Type != ResultOK {
		return fmt.Errorf("Can't unpublish object at '%s'", path)
	}
	delete(p.objectMap, path)
	return nil
}

func (p *Plugin) ID() string {
	return p.id
}

func (p *Plugin) Call(path, methodName string, params ...interface{}) ([]interface{}, error) {
	sessionID := p.sessions.getUniqueSessionID()
	if obj, ok := p.objectMap[path]; ok {
		return obj.Call(methodName, params...)
	}
	data, err := archiveMethodCallMessage(CallMethod, sessionID, path, methodName, params)
	if err != nil {
		return nil, err
	}
	p.socket.Write(data)
	message := p.sessions.receiveAndClose(sessionID)
	var result methodCall
	err = json.Unmarshal(message.body, &result)
	if err != nil {
		return nil, err
	}
	return result.Params, nil
}

func (p *Plugin) RunLoop() (err error) {
	for {
		err = p.receiveMessage()
		if err != nil {
			break
		}
	}
	return
}
