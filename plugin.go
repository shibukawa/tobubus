package tobubus

import (
	"errors"
	"fmt"
	"github.com/k0kubun/pp"
	"github.com/shibukawa/localsocket"
	"log"
	"net"
	"sync"
)

type Plugin struct {
	pipeName  string
	id        string
	socket    net.Conn
	connected bool
	sessions  *sessionManager
	lock      sync.RWMutex

	objectMap map[string]*Proxy
}

// NewPlugin creates Plugin instance.
//
// First argument is pipe name.
// ${TMP}/<pipename> for unix domain socket.
// \\.\pipe\<pipename> for Windows named pipe.
//
// Second argument is a ID of plugin
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
		sessions:  newSessionManager(recycleStrategy),
	}, nil
}

// Register methods notifies to host that plugin is ready to work
func (p *Plugin) Connect() (err error) {
	if p.socket == nil {
		return errors.New("Socket is already closed")
	}
	go func() {
		for {
			err = p.receiveMessage()
			if err != nil {
				break
			}
		}
	}()
	err = p.connect()
	if err != nil {
		p.Close()
		return err
	}
	return
}

// Register methods notifies to host that plugin is ready to work
func (p *Plugin) ConnectAndServe() (err error) {
	if p.socket == nil {
		return errors.New("Socket is already closed")
	}
	wait := make(chan error)
	go func() {
		for {
			err = p.receiveMessage()
			if err != nil {
				wait <- err
				break
			}
		}
	}()
	err = p.connect()
	if err != nil {
		p.Close()
		return err
	}
	return <-wait
}

// Unregister methods notifies to host that plugin is not working anymore.
func (p *Plugin) Close() error {
	socket := p.socket
	if socket == nil {
		return errors.New("Socket is already closed")
	}
	sessionID := p.sessions.getUniqueSessionID()
	socket.Write(archiveMessage(CloseClient, sessionID, nil))
	message := p.sessions.receiveAndClose(sessionID)
	err := socket.Close()
	if err != nil {
		return err
	}
	p.socket = nil
	if message.Type != ResultOK {
		return fmt.Errorf("Unregister error: '%s'", p.pipeName)
	}
	return nil
}

func (p *Plugin) ConfirmPath(path string) bool {
	if p.socket == nil {
		return false
	}
	sessionID := p.sessions.getUniqueSessionID()
	p.socket.Write(archiveMessage(ConfirmPath, sessionID, []byte(path)))
	message := p.sessions.receiveAndClose(sessionID)
	return message.Type == ResultOK
}

func (p *Plugin) Publish(path string, service interface{}) error {
	if p.socket == nil {
		return errors.New("Socket is already closed")
	}
	if p.connected {
		return errors.New("Plugin is already connected to host")
	}
	proxy, err := NewProxy(service)
	if err != nil {
		return err
	}
	p.lock.Lock()
	defer p.lock.Unlock()
	p.objectMap[path] = proxy
	return nil
}

func (p *Plugin) ID() string {
	return p.id
}

func (p *Plugin) Call(path, methodName string, params ...interface{}) ([]interface{}, error) {
	if p.socket == nil {
		return nil, errors.New("Socket is already closed")
	}
	p.lock.RLock()
	obj, ok := p.objectMap[path]
	p.lock.RUnlock()
	if ok {
		return obj.Call(methodName, params...)
	}
	sessionID := p.sessions.getUniqueSessionID()
	data, err := archiveMethodCallMessage(CallMethod, sessionID, path, methodName, params)
	if err != nil {
		return nil, err
	}
	_, err = p.socket.Write(data)
	if err != nil {
		return nil, err
	}
	message := p.sessions.receiveAndClose(sessionID)
	result := parseMethodCallMessage(message.body)
	return result.Params, nil
}

func (p *Plugin) receiveMessage() error {
	if p.socket == nil {
		return errors.New("Socket is already closed")
	}
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
			method := parseMethodCallMessage(msg.body)
			p.lock.RLock()
			obj, ok := p.objectMap[method.Path]
			p.lock.RUnlock()
			if !ok {
				p.socket.Write(archiveMessage(ResultObjectNotFound, msg.ID, nil))
				return
			}
			defer func() {
				err := recover()
				if err != nil {
					log.Printf("Remote Method Call Error: msgID: %d path: '%s' method: '%s'\n", msg.ID, method.Path, method.Method)
					log.Printf("Params: %s\n", pp.Sprint(method.Params))
					log.Printf("Error Detail: %v\n", err)
					p.socket.Write(archiveMessage(ResultMethodError, msg.ID, nil))
				}
			}()
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
	case CloseClient:
		socket := p.socket
		p.socket = nil
		socket.Write(archiveMessage(ResultOK, msg.ID, nil))
		err := socket.Close()
		if err != nil {
			return err
		}
		return errors.New("socket closed")
	case ConfirmPath:
		p.socket.Write(archiveMessage(ResultNG, msg.ID, nil))
	case ConnectClient:
		p.socket.Write(archiveMessage(ResultNG, msg.ID, nil))
	}
	return nil
}

func (p *Plugin) connect() error {
	sessionID := p.sessions.getUniqueSessionID()
	p.socket.Write(archiveMessage(ConnectClient, sessionID, []byte(p.id)))
	message := p.sessions.receiveAndClose(sessionID)
	if message.Type != ResultOK {
		p.socket.Close()
		return fmt.Errorf("Can't connect to '%s'", p.pipeName)
	}
	for path, proxy := range p.objectMap {
		err := p.publish(path, proxy)
		if err != nil {
			p.socket.Close()
			return err
		}
	}
	p.connected = true
	return nil
}

func (p *Plugin) publish(path string, proxy *Proxy) error {
	sessionID := p.sessions.getUniqueSessionID()
	p.socket.Write(archiveMessage(Publish, sessionID, []byte(path)))
	message := p.sessions.receiveAndClose(sessionID)
	if message.Type != ResultOK {
		return fmt.Errorf("Can't publish object at '%s'", path)
	}
	return nil
}
