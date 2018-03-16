package main // import "github.com/brettneese/go-poller"

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	jmespath "github.com/jmespath/go-jmespath"
	viper "github.com/spf13/viper"
)

var svc *s3.S3
var httpClient http.Client

// https://groups.google.com/forum/#!topic/golang-nuts/W1KJQr35NE0
func doEvery(d time.Duration, f func(time.Time)) {
	for x := range time.Tick(d) {
		f(x)
	}
}

func createBucketIfNeeded() {
	// https://docs.aws.amazon.com/sdk-for-go/api/service/s3/#S3.CreateBucket

	input := &s3.CreateBucketInput{
		Bucket: aws.String(viper.GetString("S3_BUCKET")),
	}

	result, err := svc.CreateBucket(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case s3.ErrCodeBucketAlreadyExists:
				log.Info(s3.ErrCodeBucketAlreadyExists, aerr.Error())
			case s3.ErrCodeBucketAlreadyOwnedByYou:
				log.Error(s3.ErrCodeBucketAlreadyOwnedByYou, aerr.Error())
			default:
				log.Error(aerr.Error())
			}
		} else {
			log.Error(err.Error())
		}
		return
	}

	log.Info(result)
}

func objectExists(filename string) bool {
	// https://docs.aws.amazon.com/sdk-for-go/api/service/s3/#example_S3_GetObject_shared00

	input := &s3.GetObjectInput{
		Bucket: aws.String(viper.GetString("S3_BUCKET")),
		Key:    aws.String(filename),
	}

	result, err := svc.GetObject(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case s3.ErrCodeNoSuchBucket:
				log.Error(s3.ErrCodeNoSuchBucket, aerr.Error())
			case s3.ErrCodeNoSuchKey:
				log.Debug(s3.ErrCodeNoSuchKey, aerr.Error())
			default:
				log.Error(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}

		return false

	}

	if result != nil {
		log.Debug("Object already exists: ", filename)

		return true
	}

	return false
}

func getData(t time.Time) {
	var response interface{}

	url := viper.GetString("PROVIDER_API_ROOT")
	jmespathExpression := viper.GetString("PROVIDER_JMESPATH_EXPRESSION")

	req, err := http.NewRequest(http.MethodGet, url, nil)
	req.Header.Add("Connection", "close")

	if err != nil {
		log.Fatal(err)
	}

	res, getErr := httpClient.Do(req)

	if res != nil {
		defer res.Body.Close()
	}

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
	filteredData, jmesErr := jmespath.Search(jmespathExpression, response)

	if jmesErr != nil {
		log.Fatal(jmesErr)
	}

	saveData(filteredData)
}

func saveData(jsonData interface{}) {

	fileJSON, _ := json.Marshal(jsonData)
	fileMd5 := md5.Sum(fileJSON)

	filename := hex.EncodeToString(fileMd5[:])

	if objectExists(filename) == false {
		writeObject(filename, fileJSON)
	}
}

func writeObject(filename string, fileJSON []byte) {
	// https: //docs.aws.amazon.com/sdk-for-go/api/service/s3/#example_S3_PutObject_shared00

	input := &s3.PutObjectInput{
		Body:   bytes.NewReader(fileJSON),
		Bucket: aws.String(viper.GetString("S3_BUCKET")),
		Key:    aws.String(filename),
	}

	result, err := svc.PutObject(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
		return
	}

	if result != nil {
		log.Info("Writing object: ", filename)
	}

	log.Debug(result)
}

func init() {

	svc = s3.New(session.Must(session.NewSession(&aws.Config{
		Region: aws.String("us-east-1"),
	})))

	// https://blog.alexellis.io/golang-json-api-client/
	httpClient = http.Client{
		Timeout: time.Second * viper.GetDuration("HTTP_TIMEOUT"),
	}

}

func main() {
	log.SetLevel(log.DebugLevel)

	viper.AutomaticEnv()

	// set some default variables
	viper.SetDefault("HTTP_TIMEOUT", 5000)
	viper.SetDefault("PROVIDER_REFRESH_INTERVAL", 5000)
	viper.SetDefault("PROVIDER_JMESPATH_EXPRESSION", "*")
	viper.SetDefault("STAGE", "staging")
	viper.SetDefault("PROJECT_ID", "org.opentransit.pollerv2")

	// if the env var S3_BUCKET is set, use that value, otherwise compute a bucket name from the STAGE
	if viper.GetString("S3_BUCKET") == "" && viper.GetString("STAGE") != "" && viper.GetString("PROVIDER_ID") != "" {
		bucketName := viper.GetString("PROJECT_ID") + "." + viper.GetString("STAGE") + "." + viper.GetString("PROVIDER_ID")
		viper.Set("S3_BUCKET", bucketName)
	} else {
		log.Error("Please supply either an S3_BUCKET environment variable; or a STAGE and PROVIDER_ID environment variable.")
	}

	createBucketIfNeeded()

	doEvery(viper.GetDuration("PROVIDER_REFRESH_INTERVAL")*time.Millisecond, getData)
}
