package tobubus

import (
	"github.com/shibukawa/localsocket"
	"github.com/shibukawa/mockconn"
	"net"
	"testing"
)

type testStruct struct {
	args   []string
	result string
}

func (ts *testStruct) TestMethod(arg string) string {
	ts.args = []string{arg}
	return ts.result
}

func (ts *testStruct) testMethod(arg string) string {
	ts.args = []string{arg}
	return ts.result
}

func newPluginForTest(pipeName, id string, t *testing.T) (*Plugin, *mockconn.Conn) {
	socket := mockconn.New(t)
	return &Plugin{
		pipeName:  pipeName,
		id:        id,
		socket:    socket,
		objectMap: make(map[string]*Proxy),
		sessions:  newSessionManager(incrementStrategy),
	}, socket
}

func newHostForTest(pipeName string) *Host {
	server := localsocket.NewLocalServer(pipeName)
	host := &Host{
		server:               server,
		sessions:             newSessionManager(incrementStrategy),
		pluginReservedSpaces: make(map[string]net.Conn),
		localObjectMap:       make(map[string]*Proxy),
		sockets:              make(map[string]net.Conn),
	}
	server.SetOnConnectionCallback(func(socket net.Conn) {
		go host.listenAndServeTo(socket)
	})
	return host
}
