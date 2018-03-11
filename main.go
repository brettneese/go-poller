package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	v "github.com/spf13/viper"
)

type JSONResponse map[string]interface{}

// https://groups.google.com/forum/#!topic/golang-nuts/W1KJQr35NE0
func doEvery(d time.Duration, f func(time.Time)) {
	for x := range time.Tick(d) {
		f(x)
	}
}

func getJson(url string, target interface{}) error {
	var httpClient = &http.Client{Timeout: 10 * time.Second}

	r, err := httpClient.Get(url)
	if err != nil {
		return err
	}
	defer r.Body.Close()

	return json.NewDecoder(r.Body).Decode(target)
}

func helloworld(t time.Time) {

	resp := new(JSONResponse) // or &Foo{}

	getJson(v.GetString("PROVIDER_URL"), resp)

	fmt.Printf("%v", resp)
}

func main() {
	v.AutomaticEnv()

	doEvery(v.GetDuration("REFRESH_INTERVAL")*time.Millisecond, helloworld)
}
