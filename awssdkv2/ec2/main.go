package main

// Use this code snippet in your app.
// If you need more information about configurations or implementing the sample code, visit the AWS docs:
// https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/setting-up.html

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/aws/aws-lambda-go/events"
	runtime "github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
)

type TerminationDetail struct {
	LifecycleActionToken string `json:"LifecycleActionToken"`
	AutoScalingGroupName string `json:"AutoScalingGroupName"`
	LifecycleHookName    string `json:"LifecycleHookName"`
	EC2InstanceId        string `json:"EC2InstanceId"`
	LifecycleTransition  string `json:"LifecycleTransition"`
}

var ccfg aws.Config

func getSecret(ctx context.Context, event events.CloudWatchEvent) {
	
	terminationdetail := TerminationDetail{}

	err := json.Unmarshal(event.Detail, &terminationdetail)

	if err != nil {
		log.Printf("Unmarshal Error: %s", err)
	} else {
		log.Printf("Event: %s", terminationdetail)
	}
}

// ...

func init() {
	var err error
	region := "ap-northeast-2"

	//Create a Secrets Manager client
	ccfg, err = config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
	)
	if err != nil {
		// handle error
		fmt.Println(err.Error())
	}

}

func main() {
	runtime.Start(getSecret)
}
