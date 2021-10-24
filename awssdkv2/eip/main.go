package main

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

func deleteARecord(ctx context.Context) {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("ap-northeast-2"))
	if err != nil {
		log.Fatalf("failed to load configuration, %v", err)
	}

	client := ec2.NewFromConfig(cfg)

	var ipadd []string
	ipadd = make([]string, 0)
	ipadd = append(ipadd, "3.37.216.0")

	result, err := client.DescribeAddresses(ctx, &ec2.DescribeAddressesInput{
		PublicIps: ipadd,
	})
	if err != nil {
		fmt.Println("get ip address error")
		fmt.Println(err.Error())
	}
	var deleteresult *ec2.ReleaseAddressOutput
	var deleteerr error
	for _, set := range result.Addresses {
		deleteresult, deleteerr = client.ReleaseAddress(ctx, &ec2.ReleaseAddressInput{
			AllocationId: set.AllocationId,
		})
		if deleteerr != nil {
			fmt.Println("delete err")
			fmt.Println(deleteerr.Error())
		}
	}

	fmt.Println(deleteresult.ResultMetadata)

}

func main() {
	lambda.Start(deleteARecord)
}
