package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	jmespath "github.com/jmespath/go-jmespath"
	viper "github.com/spf13/viper"
)

type JSONBlob []byte

// https://groups.google.com/forum/#!topic/golang-nuts/W1KJQr35NE0
func doEvery(d time.Duration, f func(time.Time)) {
	for x := range time.Tick(d) {
		f(x)
	}
}

func getData(t time.Time) {
	var response interface{}

	url := viper.GetString("PROVIDER_URL")

	httpClient := http.Client{
		Timeout: time.Second * 10, // Maximum of 2 secs
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Fatal(err)
	}

	res, getErr := httpClient.Do(req)
	if getErr != nil {
		log.Fatal(getErr)
	}

	body, readErr := ioutil.ReadAll(res.Body)
	if readErr != nil {
		log.Fatal(readErr)
	}

	jsonErr := json.Unmarshal(body, &response)
	if jsonErr != nil {
		log.Fatal(jsonErr)
	}

	processData(response)
}

func processData(jsonBlob interface{}) {

	v, err := jmespath.Search("ctatt.[route]", jsonBlob)

	if err != nil {
		log.Fatal(err)
	}
	log.Println(v)
}
func main() {
	viper.AutomaticEnv()

	doEvery(viper.GetDuration("REFRESH_INTERVAL")*time.Millisecond, getData)
}
