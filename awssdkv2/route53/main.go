package main

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"
)

func deleteARecord(ctx context.Context) {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("ap-northeast-2"))
	if err != nil {
		log.Fatalf("failed to load configuration, %v", err)
	}

	client := route53.NewFromConfig(cfg)
	var zoneid, hostid string
	zoneid = "Z0038228146866JXF6J9I"
	hostid = "www.jhoh-test1075.com"

	rrset, err := client.ListResourceRecordSets(ctx, &route53.ListResourceRecordSetsInput{
		HostedZoneId:    &zoneid,
		StartRecordName: &hostid,
		StartRecordType: types.RRTypeA,
	})
	if err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Println("success")
	}

	fmt.Println(*rrset)
	fmt.Println(rrset.ResultMetadata)

	var change []types.Change
	change = make([]types.Change, 0)

	for _, set := range rrset.ResourceRecordSets {
		fmt.Println(len(set.ResourceRecords))
		fmt.Println(set.ResourceRecords)
		fmt.Println(*set.ResourceRecords[0].Value)
		change = append(change, types.Change{
			Action:            types.ChangeActionDelete,
			ResourceRecordSet: &set,
		})
	}
	fmt.Println(len(change))

	input := &route53.ChangeResourceRecordSetsInput{
		ChangeBatch: &types.ChangeBatch{
			Changes: change,
		},
		HostedZoneId: &zoneid,
	}

	output, err := client.ChangeResourceRecordSets(ctx, input)
	if err != nil {
		fmt.Println("delete error")
		fmt.Println(err.Error())
	}
	fmt.Println(*output)

}

func main() {
	lambda.Start(deleteARecord)
}
