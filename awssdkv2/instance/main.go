package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchevents"
)

type CloudWatchEventDetails struct {
	EventVersion     string    `json:"eventVersion"`
	EventID          string    `json:"eventID"`
	EventTime        time.Time `json:"eventTime"`
	EventType        string    `json:"eventType"`
	ResponseElements struct {
		OwnerID      string `json:"ownerId"`
		InstancesSet struct {
			Items []struct {
				InstanceID string `json:"instanceId"`
			} `json:"items"`
		} `json:"instancesSet"`
	} `json:"responseElements"`
	AwsRegion    string `json:"awsRegion"`
	EventName    string `json:"eventName"`
	UserIdentity struct {
		UserName    string `json:"userName"`
		PrincipalID string `json:"principalId"`
		AccessKeyID string `json:"accessKeyId"`
		InvokedBy   string `json:"invokedBy"`
		Type        string `json:"type"`
		Arn         string `json:"arn"`
		AccountID   string `json:"accountId"`
	} `json:"userIdentity"`
	EventSource string `json:"eventSource"`
}

var eventDetail map[string]interface{}

var eventDetails CustomAwsLambdaEvent.CloudWatchEventDetails

func main() {

	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("ap-northeast-2"))
	if err != nil {
		log.Fatalf("failed to load configuration, %v", err)
	}

	client := cloudwatchevents.NewFromConfig(cfg)

	cloudwatchevents.Event

	client.ListEvent
}

func HandleRequest(ctx context.Context, jsonEvent events.CloudWatchEvent) {

	//Unmarshall the CloudWatchEvent Struct Details
	err := json.Unmarshal(jsonEvent.Detail, &eventDetails)
	if err != nil {
		log.Fatal("Could not unmarshal scheduled event: ", err)
		fmt.Println("Could not unmarshal scheduled event: ", err)
	}

	outputJSON, err := json.Marshal(eventDetails)
	if err != nil {
		log.Fatal("Could not unmarshal scheduled event: ", err)
		fmt.Println("Could not unmarshal scheduled event: ", err)
	}

	fmt.Println("This is the JSON for event details", string(outputJSON))

	version := jsonEvent.Version
	id := jsonEvent.ID
	detailType := jsonEvent.DetailType
	source := jsonEvent.Source
	accountId := jsonEvent.AccountID
	eventTime := jsonEvent.Time
	region := jsonEvent.Region
	resources := jsonEvent.Resources

	//eventname = detail['eventName']
	eventName := eventDetails.EventName

	//arn = detail['userIdentity']['arn']
	arn := eventDetails.UserIdentity.Arn

	//principal = detail['userIdentity']['principalId']
	principal := eventDetails.UserIdentity.PrincipalID

	//userType = detail['userIdentity']['type']
	userType := eventDetails.UserIdentity.Type

	//user = detail['userIdentity']['userName']
	user := eventDetails.UserIdentity.UserName

	//date = detail['eventTime']
	date := eventDetails.EventTime

	//items = detail['responseElements']['instancesSet']['items'][0]['instanceId']
	instanceId := eventDetails.ResponseElements.InstancesSet.Items[0].InstanceID

	fmt.Println("This is the version: ", version)
	fmt.Println("This is the id: ", id)
	fmt.Println("This is the detailType: ", detailType)
	fmt.Println("This is the Source: ", source)
	fmt.Println("This is the accountId: ", accountId)
	fmt.Println("This is the eventTime: ", eventTime)
	fmt.Println("This is the region: ", region)
	fmt.Println("This is the resource: ", resources)

	fmt.Println("Here is the eventName: ", eventName)
	fmt.Println("Here is the date: ", date)
	fmt.Println("Here is the arn: ", arn)
	fmt.Println("Here is the principalId: ", principal)
	fmt.Println("Here is the userType: ", userType)
	fmt.Println("Here is the user: ", user)
	fmt.Println("Here is the instanceId: ", instanceId)

}
