package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
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
		Timeout: time.Second * viper.GetDuration("HTTP_TIMEOUT"),
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
	//start an s3 session
	svc := s3.New(session.Must(session.NewSession(&aws.Config{
		Region: aws.String("us-east-1"),
	})))

	var filename string

	fileJSON, _ := json.Marshal(jsonData)
	fileMd5 := md5.Sum(fileJSON)

	filename = hex.EncodeToString(fileMd5[:])

	err := ioutil.WriteFile(filename, fileJSON, 0644)

	if err != nil {
		log.Fatal(err)
	}

	if objectExists(svc, filename) == false {
		writeObject(svc, filename, jsonData)
	}
}

func objectExists(svc *s3.S3, filename string) bool {
	// https://docs.aws.amazon.com/sdk-for-go/api/service/s3/#example_S3_GetObject_shared00
	input := &s3.GetObjectInput{
		Bucket: aws.String("com.brettneese.opentransit-pollerv2.production.cta-train"),
		Key:    aws.String(filename),
	}

	result, err := svc.GetObject(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case s3.ErrCodeNoSuchBucket:
				// makeBucket()
			case s3.ErrCodeNoSuchKey:
				fmt.Println(s3.ErrCodeNoSuchKey, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return false
	}

	fmt.Println(result)

	return true
}

func main() {
	viper.AutomaticEnv()

	viper.SetDefault("HTTP_TIMEOUT", 5000)
	viper.SetDefault("REFRESH_INTERVAL", 5000)
	viper.SetDefault("JMESPATH_EXPRESSION", "*")

	doEvery(viper.GetDuration("REFRESH_INTERVAL")*time.Millisecond, getData)
}
