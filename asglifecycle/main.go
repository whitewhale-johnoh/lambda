package main

import (
	"context"
	"fmt"
	"log"

	config "github.com/aws/aws-sdk-go-v2/config"
)

type TerminationDetail struct {
	LifecycleActionToken string `json:"LifecycleActionToken"`
	AutoScalingGroupName string `json:"AutoScalingGroup"`
	LifecycleHookName    string `json:"LifecycleHookName"`
	EC2InstanceId        string `json:"EC2InstanceId"`
	LifecycleTransition  string `json:"LifecycleTransition"`
}

func init() {

}

func main() {
	getConfig()
}

func getConfig() {

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(cfg)

}
