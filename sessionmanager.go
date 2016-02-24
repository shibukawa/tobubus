package tobubus

import (
	"math"
	"sync"
)

type sessionManager struct {
	lock     sync.RWMutex
	sessions map[uint32]chan *message
}

func newSessionManager() *sessionManager {
	return &sessionManager{
		sessions: make(map[uint32]chan *message),
	}
}

func (g *sessionManager) getUniqueSessionID() uint32 {
	g.lock.Lock()
	defer g.lock.Unlock()
	var id uint32
	for id = 0; id < math.MaxUint32; id++ {
		if _, ok := g.sessions[id]; !ok {
			g.sessions[id] = make(chan *message)
			return id
		}
	}
	panic("id error")
}

func (g *sessionManager) receiveAndClose(id uint32) *message {
	g.lock.Lock()
	channel, ok := g.sessions[id]
	if !ok {
		channel = make(chan *message)
		g.sessions[id] = channel
	}
	g.lock.Unlock()
	result := <-channel
	delete(g.sessions, id)
	return result
}

func (g *sessionManager) getChannelOfSessionID(id uint32) chan *message {
	g.lock.Lock()
	defer g.lock.Unlock()
	if channel, ok := g.sessions[id]; ok {
		return channel
	}
	channel := make(chan *message)
	g.sessions[id] = channel
	return channel
}
