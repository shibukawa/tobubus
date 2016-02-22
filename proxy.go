package tobubus

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

type Proxy struct {
	instance       interface{}
	methods        map[string]reflect.Value
	privateMethods map[string]bool
}

func hasUpperPrefix(name string) bool {
	prefix := name[:1]
	return prefix == strings.ToUpper(prefix)
}

func NewProxy(instance interface{}) (*Proxy, error) {
	if instance == nil {
		return nil, errors.New("can't register nil")
	}
	proxy := &Proxy{
		instance: instance,
		methods:  make(map[string]reflect.Value),
	}
	v := reflect.ValueOf(instance)
	t := v.Type()
	n := t.NumMethod()
	for i := 0; i < n; i++ {
		name := t.Method(i).Name
		if hasUpperPrefix(name) {
			proxy.methods[name] = v.MethodByName(name)
		} else {
			proxy.privateMethods[name] = true
		}
	}

	return proxy, nil
}

func (p *Proxy) Call(name string, args ...interface{}) ([]interface{}, error) {
	method, ok := p.methods[name]
	if !ok {
		if p.privateMethods[name] {
			return nil, fmt.Errorf("Method '%s' is private", name)
		} else {
			return nil, fmt.Errorf("Method '%s' is undefined", name)
		}
	}
	newArgs := make([]reflect.Value, len(args))
	for i, arg := range args {
		newArgs[i] = reflect.ValueOf(arg)
	}
	results := method.Call(newArgs)
	newResults := make([]interface{}, len(results))
	for i, result := range results {
		newResults[i] = result.Interface()
	}
	return newResults, nil
}
