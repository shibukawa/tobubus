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
		sessions:             newSessionManager(recycleStrategy),
		pluginReservedSpaces: make(map[string]net.Conn),
		localObjectMap:       make(map[string]*Proxy),
		sockets:              make(map[string]net.Conn),
	}
	server.SetOnConnectionCallback(func(socket net.Conn) {
		go host.listenAndServeTo(socket)
	})
	return host
}

func (h *Host) GetSocket(pluginID string) net.Conn {
	return h.sockets[pluginID]
}

func (h *Host) GetPluginID(pluginSocket net.Conn) string {
	h.lock.RLock()
	defer h.lock.RUnlock()
	for id, socket := range h.sockets {
		if socket == pluginSocket {
			return id
		}
	}
	return ""
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

func (h *Host) Close() error {
	for _, socket := range h.sockets {
		socket.Close()
	}
	h.server.Close()
	return nil
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
	socket.Write(archiveMessage(CloseClient, sessionID, nil))
	message := h.sessions.receiveAndClose(sessionID)
	socket.Close()
	if message.Type != ResultOK {
		return fmt.Errorf("Unregister error: '%s'", pluginID)
	}
	return nil
}

func (h *Host) Publish(path string, service interface{}) error {
	proxy, err := NewProxy(service)
	if err != nil {
		return err
	}
	h.lock.Lock()
	defer h.lock.Unlock()
	h.localObjectMap[path] = proxy
	return nil
}

func (h *Host) Unpublish(path string) error {
	h.lock.Lock()
	defer h.lock.Unlock()
	_, ok := h.localObjectMap[path]
	if ok {
		delete(h.localObjectMap, path)
		return nil
	}
	return fmt.Errorf("Unpublish error: no object is registered at '%s'", path)
}

func (h *Host) Call(path, methodName string, params ...interface{}) ([]interface{}, error) {
	h.lock.RLock()
	obj, ok := h.localObjectMap[path]
	if ok {
		h.lock.RUnlock()
		return obj.Call(methodName, params...)
	} else {
		socket, ok := h.pluginReservedSpaces[path]
		h.lock.RUnlock()
		if ok {
			sessionID := h.sessions.getUniqueSessionID()
			data, err := archiveMethodCallMessage(CallMethod, sessionID, path, methodName, params)
			if err != nil {
				return nil, err
			}
			_, err = socket.Write(data)
			if err != nil {
				return nil, err
			}
			message := h.sessions.receiveAndClose(sessionID)
			var result methodCall
			err = json.Unmarshal(message.body, &result)
			if err != nil {
				return nil, err
			}
			return result.Params, nil
		}
		return nil, fmt.Errorf("There is no object in path '%s'.", path)
	}

	return nil, nil
}

func (h *Host) ConfirmPath(path string) bool {
	_, ok := h.localObjectMap[path]
	if ok {
		return true
	}
	_, ok = h.pluginReservedSpaces[path]
	return ok
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
	case ConnectClient:
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
	case Publish:
		path := string(msg.body)
		h.lock.Lock()
		_, ok := h.pluginReservedSpaces[path]
		if ok {
			h.lock.Unlock()
			socket.Write(archiveMessage(ResultNG, msg.ID, nil))
		} else {
			socket.Write(archiveMessage(ResultOK, msg.ID, nil))
			h.pluginReservedSpaces[path] = socket
			h.lock.Unlock()
		}
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
	case CloseClient:
		socketID := h.GetPluginID(socket)
		if socketID == "" {
			socket.Write(archiveMessage(ResultNG, msg.ID, nil))
		} else {
			h.lock.Lock()
			var removeTargetPaths []string
			for path, mappedSocket := range h.pluginReservedSpaces {
				if mappedSocket == socket {
					removeTargetPaths = append(removeTargetPaths, path)
				}
			}
			for _, removeTargetPath := range removeTargetPaths {
				delete(h.pluginReservedSpaces, removeTargetPath)
			}
			delete(h.sockets, socketID)
			h.lock.Unlock()
			socket.Write(archiveMessage(ResultOK, msg.ID, nil))
		}
	case ConfirmPath:
		_, ok := h.localObjectMap[string(msg.body)]
		if ok {
			socket.Write(archiveMessage(ResultOK, msg.ID, nil))
		} else {
			socket.Write(archiveMessage(ResultObjectNotFound, msg.ID, nil))
		}
	}
	return nil
}
