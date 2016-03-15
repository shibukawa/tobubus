package tobubus_test

import (
	"fmt"
	"github.com/shibukawa/tobubus"
)

type Calculator struct {
}

func (c Calculator) Fib(n int64) int64 {
	if n < 2 {
		return n
	}
	return c.Fib(n-2) + c.Fib(n-1)
}

func plugin(wait chan string) {
	// First argument is pipe name
	// Second argument is a ID of plugin
	plugin, err := tobubus.NewPlugin("tobubus.test", "github.com/shibukawa/tobubus/example")
	if err != nil {
		wait <- "error"
	}
	// Start communication with host and start event loop
	plugin.Publish("/calculator", &Calculator{})
	err = plugin.Connect()
	// Publish object. Now host can call this instance's method.
	wait <- "ready client"
	<-wait // wait finish
	plugin.Close()
}

func Example() {
	wait := make(chan string)

	host := tobubus.NewHost("tobubus.test")
	host.Listen()
	defer host.Close()

	go plugin(wait)
	pStatus := <-wait // ready client
	if pStatus == "error" {
		return
	}

	// Call plugin's method
	results, err := host.Call("/calculator", "Fib", int(10))
	if err != nil {
		fmt.Println(err.Error())
	}
	fmt.Println(results[0].(int64))
	// Output:
	// 55
	wait <- "finish"
}
