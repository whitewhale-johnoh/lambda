package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	r53type "github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	smtype "github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
)

var cfg aws.Config
var hostzone Hostzone
var mutex = new(sync.Mutex)

type Hostzone struct {
	Id       string `json:"HostZoneID"`
	Hostname string `json:"Hostname"`
}

type TerminationDetail struct {
	LifecycleActionToken string `json:"LifecycleActionToken"`
	AutoScalingGroupName string `json:"AutoScalingGroupName"`
	LifecycleHookName    string `json:"LifecycleHookName"`
	EC2InstanceId        string `json:"EC2InstanceId"`
	LifecycleTransition  string `json:"LifecycleTransition"`
}

func listEIP(ctx context.Context, client *ec2.Client, ip string) *ec2.DescribeAddressesOutput {

	var ipadd []string
	ipadd = make([]string, 0, 1000)
	ipadd = append(ipadd, ip)
	log.Printf("ipadd length: %d", len(ipadd))

	result, err := client.DescribeAddresses(ctx, &ec2.DescribeAddressesInput{
		PublicIps: ipadd,
	})
	if err != nil {
		log.Printf("Describe Address Error: %s", err.Error())
		return nil
	} else {
		log.Printf("Described Address Successfully")
		return result
	}
}

func releaseEIP(ctx context.Context, client *ec2.Client, list *ec2.DescribeAddressesOutput) {
	var deleteresult *ec2.ReleaseAddressOutput
	var deleteerr error
	log.Printf("result address len: %d", len(list.Addresses))
	for _, set := range list.Addresses {
		deleteresult, deleteerr = client.ReleaseAddress(ctx, &ec2.ReleaseAddressInput{
			AllocationId: set.AllocationId,
		})
		if deleteerr != nil {
			log.Printf("Release Address Error: %s", deleteerr.Error())
		} else {
			log.Printf("Release Address Successfully")
		}
	}

	fmt.Println(deleteresult.ResultMetadata)
}

func cleanupEIP(ctx context.Context, ip string) {
	eipclient := ec2.NewFromConfig(cfg)
	result := listEIP(ctx, eipclient, ip)
	if result != nil {
		releaseEIP(ctx, eipclient, result)
	}
}

func listDNSRecord(ctx context.Context, client *route53.Client, hostzone Hostzone) (*route53.ListResourceRecordSetsOutput, error) {
	rrset, err := client.ListResourceRecordSets(ctx, &route53.ListResourceRecordSetsInput{
		HostedZoneId:    &hostzone.Id,
		StartRecordName: &hostzone.Hostname,
		StartRecordType: r53type.RRTypeA,
	})
	if err != nil {
		log.Printf("List Resource Record Set error: %s", err.Error())
		return nil, err
	} else {
		log.Printf("success")
		return rrset, nil
	}
}

func prepareDNSRecord(ctx context.Context, lo *route53.ListResourceRecordSetsOutput, hostzone Hostzone) (route53.ChangeResourceRecordSetsInput, string) {
	change := make([]r53type.Change, 0, 100)
	var ip string
	var input route53.ChangeResourceRecordSetsInput
	for _, set := range lo.ResourceRecordSets {

		ip = *set.ResourceRecords[0].Value
		//log.Printf("length of Resource Records: %d", len(set.ResourceRecords))
		log.Printf("IP address: %s", *set.ResourceRecords[0].Value)
		change = append(change, r53type.Change{
			Action:            r53type.ChangeActionDelete,
			ResourceRecordSet: &set,
		})

	}

	input = route53.ChangeResourceRecordSetsInput{
		ChangeBatch: &r53type.ChangeBatch{
			Changes: change,
		},
		HostedZoneId: &hostzone.Id,
	}

	log.Printf("change length: %d", len(input.ChangeBatch.Changes))

	return input, ip

}

func deleteDNSRecord(ctx context.Context, client *route53.Client, input route53.ChangeResourceRecordSetsInput) *route53.ChangeResourceRecordSetsOutput {
	output, err := client.ChangeResourceRecordSets(ctx, &input)
	if err != nil {
		log.Printf("Delete Resource Record Set Error: %s", err.Error())
		return nil
	} else {
		log.Printf("Successfully deleted")
		return output
	}
}

func cleanupRoute53(ctx context.Context, td TerminationDetail) string {
	var host Hostzone
	host.Id = hostzone.Id
	host.Hostname = td.EC2InstanceId + "." + hostzone.Hostname
	r53client := route53.NewFromConfig(cfg)

	rresult, err := listDNSRecord(ctx, r53client, host)
	if err != nil {
		return ""
	}

	log.Printf("rrset length: %d", len(rresult.ResourceRecordSets))
	input, ip := prepareDNSRecord(ctx, rresult, host)
	output := deleteDNSRecord(ctx, r53client, input)
	fmt.Println(output.ChangeInfo.Status)
	return ip
}

func cleanup(ctx context.Context, event events.CloudWatchEvent) {
	fmt.Println(event)
	terminationdetail := TerminationDetail{}
	fmt.Println(terminationdetail)
	err := json.Unmarshal(event.Detail, &terminationdetail)

	if err != nil {
		log.Printf("Unmarshal Error: %s", err.Error())
	} else {
		log.Printf("Event: %s", terminationdetail)
	}

	////////////////////// clean up record set in Route 53//////////////////////
	ip := cleanupRoute53(ctx, terminationdetail)
	////////////////////// Delete EIP /////////////////////////
	if ip != "" {
		cleanupEIP(ctx, ip)
	}
}

func init() {
	var cfgerr error

	cfg, cfgerr = config.LoadDefaultConfig(context.TODO(), config.WithRegion("ap-northeast-2"))
	if cfgerr != nil {
		log.Fatalf("failed to load configuration, %v", cfgerr.Error())
	}

	secretName := "lambda/hostzoneid/"

	client := secretsmanager.NewFromConfig(cfg)

	input := &secretsmanager.GetSecretValueInput{
		SecretId:     aws.String(secretName),
		VersionStage: aws.String("AWSCURRENT"), // VersionStage defaults to AWSCURRENT if unspecified
	}

	result, err := client.GetSecretValue(context.TODO(), input)
	if err != nil {
		switch err.(type) {
		case *smtype.DecryptionFailure:
			// Secrets Manager can't decrypt the protected secret text using the provided KMS key.
			fmt.Println(err.Error())

		case *smtype.InternalServiceError:
			// An error occurred on the server side.
			fmt.Println(err.Error())

		case *smtype.InvalidParameterException:
			// You provided an invalid value for a parameter.
			fmt.Println(err.Error())

		case *smtype.InvalidRequestException:
			// You provided a parameter value that is not valid for the current state of the resource.
			fmt.Println(err.Error())

		case *smtype.ResourceNotFoundException:
			// We can't find the resource that you asked for.
			fmt.Println(err.Error())
		default:

			fmt.Println(err.Error())
		}

		return
	}

	secretString := *result.SecretString
	secretvalue := []byte(secretString)
	json.Unmarshal(secretvalue, &hostzone)

}

func main() {
	lambda.Start(cleanup)
}
