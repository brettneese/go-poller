package main

import (
	"fmt"
	"time"

	v "github.com/spf13/viper"
)

// https://groups.google.com/forum/#!topic/golang-nuts/W1KJQr35NE0
func doEvery(d time.Duration, f func(time.Time)) {
	for x := range time.Tick(d) {
		f(x)
	}
}

func helloworld(t time.Time) {
	fmt.Printf("%v: Hello, World!\n", t)
}

func main() {
	v.AutomaticEnv()

	doEvery(v.GetDuration("REFRESH_INTERVAL")*time.Millisecond, helloworld)
}
