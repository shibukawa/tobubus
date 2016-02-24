package tobubus

import (
	"encoding/json"
	"fmt"
	"github.com/shibukawa/localsocket"
	"net"
	"sync"
)

type Plugin struct {
	pipeName      string
	id            string
	socket        net.Conn
	lock          sync.RWMutex
	nextMessageID uint32

	objectMap map[string]*Proxy
	session   map[uint32]chan *message
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
		session:   make(map[uint32]chan *message),
	}, nil
}

func (p *Plugin) messageID() uint32 {
	p.lock.Lock()
	defer p.lock.Unlock()

	id := p.nextMessageID
	p.nextMessageID++
	p.session[id] = make(chan *message)
	return id
}

func (p *Plugin) receiveMessage() error {
	msg, err := parseMessage(p.socket)
	if err != nil {
		return err
	}
	switch msg.Type {
	case ResultOK, ResultNG, ReturnMethod:
		p.lock.Lock()
		channel, ok := p.session[msg.ID]
		if !ok {
			channel = make(chan *message)
			p.session[msg.ID] = channel
		}
		p.lock.Unlock()
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

func (p *Plugin) waitMessage(messageID uint32) (*message, error) {
	p.lock.Lock()
	channel, ok := p.session[messageID]
	if !ok {
		channel = make(chan *message)
		p.session[messageID] = channel
	}
	p.lock.Unlock()
	result := <-channel
	delete(p.session, messageID)
	return result, nil
}

func (p *Plugin) Register() error {
	messageID := p.messageID()
	p.socket.Write(archiveMessage(RegisterClient, messageID, []byte(p.id)))
	message, err := p.waitMessage(messageID)
	if err != nil {
		return err
	}
	if message.Type != ResultOK {
		p.socket.Close()
		return fmt.Errorf("Can't connect to '%s'", p.pipeName)
	}
	return nil
}

func (p *Plugin) Unregister() error {
	messageID := p.messageID()
	p.socket.Write(archiveMessage(UnregisterClient, messageID, nil))
	message, err := p.waitMessage(messageID)
	p.socket.Close()
	p.socket = nil
	if err != nil {
		return err
	}
	if message.Type != ResultOK {
		return fmt.Errorf("Unregister error: '%s'", p.pipeName)
	}
	return nil
}

func (p *Plugin) ConfirmPath(path string) bool {
	messageID := p.messageID()
	p.socket.Write(archiveMessage(ConfirmPath, messageID, []byte(path)))
	message, err := p.waitMessage(messageID)
	if err != nil {
		return false
	}
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
	messageID := p.messageID()
	p.socket.Write(archiveMessage(Publish, messageID, []byte(path)))
	message, err := p.waitMessage(messageID)
	if err != nil {
		return err
	}
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
	messageID := p.messageID()
	p.socket.Write(archiveMessage(Unpublish, messageID, []byte(path)))
	message, err := p.waitMessage(messageID)
	if err != nil {
		return err
	}
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
	messageID := p.messageID()
	if obj, ok := p.objectMap[path]; ok {
		return obj.Call(methodName, params...)
	}
	data, err := archiveMethodCallMessage(CallMethod, messageID, path, methodName, params)
	if err != nil {
		return nil, err
	}
	p.socket.Write(data)
	message, err := p.waitMessage(messageID)
	if err != nil {
		return nil, err
	}
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
