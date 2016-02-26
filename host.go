package tobubus

import (
	"encoding/json"
	"fmt"
	"github.com/shibukawa/localsocket"
	"net"
	"sync"
)

type Host struct {
	pipeName string
	server   *localsocket.LocalServer
	sessions *sessionManager
	lock     sync.RWMutex

	pluginReservedSpaces map[string]net.Conn // path -> socket
	localObjectMap       map[string]*Proxy   // path -> proxy
	sockets              map[string]net.Conn // plugin id -> socket
}

func NewHost(pipeName string) *Host {
	server := localsocket.NewLocalServer(pipeName)
	host := &Host{
		server:               server,
		sessions:             newSessionManager(),
		pluginReservedSpaces: make(map[string]net.Conn),
		localObjectMap:       make(map[string]*Proxy),
		sockets:              make(map[string]net.Conn),
	}
	server.SetOnConnectionCallback(func(socket net.Conn) {
		go host.listenAndServeTo(socket)
	})
	return host
}

func (h *Host) Listen() error {
	return h.server.Listen()
}

func (h *Host) ListenAndServer() error {
	return h.server.ListenAndServe()
}

func (h *Host) listenAndServeTo(socket net.Conn) (err error) {
	for {
		err = h.receiveMessage(socket)
		if err != nil {
			break
		}
	}
	return
}

func (h *Host) Unregister(pluginID string) error {
	sessionID := h.sessions.getUniqueSessionID()
	h.lock.Lock()
	socket, ok := h.sockets[pluginID]
	delete(h.sockets, pluginID)
	h.lock.Unlock()
	if !ok {
		return fmt.Errorf("plugin id '%s' is not registered", pluginID)
	}
	socket.Write(archiveMessage(UnregisterClient, sessionID, nil))
	message := h.sessions.receiveAndClose(sessionID)
	socket.Close()
	if message.Type != ResultOK {
		return fmt.Errorf("Unregister error: '%s'", pluginID)
	}
	return nil
}

func (h *Host) receiveMessage(socket net.Conn) error {
	msg, err := parseMessage(socket)
	if err != nil {
		return err
	}
	switch msg.Type {
	case ResultOK, ResultNG, ReturnMethod:
		channel := h.sessions.getChannelOfSessionID(msg.ID)
		channel <- msg
	case CallMethod:
		go func() {
			method := &methodCall{}
			json.Unmarshal(msg.body, method)
			obj, ok := h.localObjectMap[method.Path]
			if !ok {
				socket.Write(archiveMessage(ResultObjectNotFound, msg.ID, nil))
				return
			}
			result, err := obj.Call(method.Method, method.Params...)
			if err != nil {
				socket.Write(archiveMessage(ResultMethodNotFound, msg.ID, nil))
			} else {
				resultMessage, err := archiveMethodCallMessage(ReturnMethod, msg.ID, "", "", result)
				if err != nil {
					socket.Write(archiveMessage(ResultNG, msg.ID, nil))
				}
				socket.Write(resultMessage)
			}
		}()
	case UnregisterClient, ConfirmPath:
	// todo
	case RegisterClient:
		pluginID := string(msg.body)
		h.lock.Lock()
		_, ok := h.sockets[pluginID]
		if ok {
			h.lock.Unlock()
			socket.Write(archiveMessage(ResultNG, msg.ID, nil))
		} else {
			socket.Write(archiveMessage(ResultOK, msg.ID, nil))
			h.sockets[pluginID] = socket
			h.lock.Unlock()
		}
	}
	return nil
}
