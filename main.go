package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	jmespath "github.com/jmespath/go-jmespath"
	viper "github.com/spf13/viper"
)

// https://groups.google.com/forum/#!topic/golang-nuts/W1KJQr35NE0
func doEvery(d time.Duration, f func(time.Time)) {
	for x := range time.Tick(d) {
		f(x)
	}
}

func getData(t time.Time) {
	var response interface{}

	url := viper.GetString("PROVIDER_URL")
	jmespathExpression := viper.GetString("JMESPATH_EXPRESSION")

	// https://blog.alexellis.io/golang-json-api-client/
	httpClient := http.Client{
		Timeout: time.Second * 5, // Maximum of 5 secs
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

	// filter out just the needed fields by supplying a jmespath expression
	// https://github.com/jmespath/go-jmespath/issues/22#issuecomment-277719772
	v, jmesErr := jmespath.Search(jmespathExpression, response)

	if jmesErr != nil {
		log.Fatal(jmesErr)
	}

	saveData(v)
}

func saveData(jsonData interface{}) {

	var filename string

	fileJSON, _ := json.Marshal(jsonData)
	fileMd5 := md5.Sum(fileJSON)

	filename = hex.EncodeToString(fileMd5[:])

	err := ioutil.WriteFile(filename, fileJSON, 0644)

	if err != nil {
		log.Fatal(err)
	}

}

func main() {
	viper.AutomaticEnv()

	viper.SetDefault("REFRESH_INTERVAL", 5000)
	viper.SetDefault("JMESPATH_EXPRESSION", "*")

	doEvery(viper.GetDuration("REFRESH_INTERVAL")*time.Millisecond, getData)
}
