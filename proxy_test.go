package tobubus

import (
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

func TestProxyCallMethod(t *testing.T) {
	obj := &testStruct{
		result: "result value",
	}
	proxy, err := NewProxy(obj)
	if err != nil {
		t.Errorf("err should be nil but: %v", err)
	}
	results, err := proxy.Call("TestMethod", "arg1")
	if err != nil {
		t.Errorf("err should be nil but: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("it should returns one result, but '%d'", len(results))
	} else if results[0] != "result value" {
		t.Errorf("results[0] should be 'test value' but '%s'", results[0])
	}
}

func TestProxyCallPrivateMethod(t *testing.T) {
	obj := &testStruct{
		result: "result value",
	}
	proxy, err := NewProxy(obj)
	if err != nil {
		t.Errorf("err should be nil but: %v", err)
	}
	_, err = proxy.Call("testMethod", "arg1")
	if err == nil {
		t.Errorf("err should nil for private method")
	}
}
