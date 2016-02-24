package tobubus

import (
	"github.com/shibukawa/mockconn"
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
		sessions:  newSessionManager(),
	}, socket
}
