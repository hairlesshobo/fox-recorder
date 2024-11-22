package shared

import (
	// "fmt"
	"os"
	"os/signal"
)

// var gotSigint chan bool

func CatchSigint(callback func()) {
	c := make(chan os.Signal, 1)
	// gotSigint = make(chan bool, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for range c {
			callback()
			// sig is a ^C, handle it
			// gotSigint <- true
		}
	}()
}

// func IsCancelRequested() bool {
// 	fmt.Println(len(gotSigint))
// 	return len(gotSigint) > 0
// }
