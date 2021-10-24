package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

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
var hostchan chan Hostzone
var dnsreousrcelistchan chan *route53.ListResourceRecordSetsOutput
var dnsdeletememberchan chan *route53.ChangeResourceRecordSetsInput
var ipchan chan string
var eiplistchan chan *ec2.DescribeAddressesOutput

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

func listEIP(ctx context.Context, client *ec2.Client) {
	log.Println("listEIP running")
	mutex.Lock()
	ip := <-ipchan
	var ipadd []string
	ipadd = make([]string, 0, 1000)
	ipadd = append(ipadd, ip)
	log.Printf("ipadd length: %d", len(ipadd))

	result, err := client.DescribeAddresses(ctx, &ec2.DescribeAddressesInput{
		PublicIps: ipadd,
	})
	if err != nil {
		log.Printf("Describe Address Error: %s", err.Error())

	} else {
		log.Printf("Described Address Successfully")
		eiplistchan <- result
	}
	mutex.Unlock()
}

func releaseEIP(ctx context.Context, client *ec2.Client) {
	log.Println("releaseEIP running")
	var deleteresult *ec2.ReleaseAddressOutput
	var deleteerr error
	list := <-eiplistchan
	log.Printf("result address len: %d", len(list.Addresses))
	for _, set := range list.Addresses {
		deleteresult, deleteerr = client.ReleaseAddress(ctx, &ec2.ReleaseAddressInput{
			AllocationId: set.AllocationId,
		})
		if deleteerr != nil {
			log.Printf("Release Address Error: %s %s", *set.PublicIp, deleteerr.Error())
		} else {
			log.Printf("Release Address Successfully: %s", *set.PublicIp)
		}
	}

	log.Printf("Deleted Result: %s", deleteresult.ResultMetadata)
}

func listDNSRecord(ctx context.Context, client *route53.Client, host Hostzone) {
	log.Println("listDNSRecord running")
	rrset, err := client.ListResourceRecordSets(ctx, &route53.ListResourceRecordSetsInput{
		HostedZoneId:    &host.Id,
		StartRecordName: &host.Hostname,
		StartRecordType: r53type.RRTypeA,
	})
	if err != nil {
		log.Printf("List Resource Record Set error: %s", err.Error())
	} else {
		log.Printf("success")
		dnsreousrcelistchan <- rrset
	}
}

func prepareDNSRecord(ctx context.Context, hostzone Hostzone) {
	log.Println("prepareDNSRecord running")
	change := make([]r53type.Change, 0, 1000)
	var input route53.ChangeResourceRecordSetsInput
	rrset := <-dnsreousrcelistchan
	go func() {
		for _, set := range rrset.ResourceRecordSets {

			mutex.Lock()
			ipchan <- *set.ResourceRecords[0].Value
			log.Printf("length of Resource Records: %d", len(set.ResourceRecords))
			log.Printf("IP address: %s", *set.ResourceRecords[0].Value)

			input = route53.ChangeResourceRecordSetsInput{
				ChangeBatch: &r53type.ChangeBatch{
					Changes: append(change, r53type.Change{
						Action:            r53type.ChangeActionDelete,
						ResourceRecordSet: &set,
					}),
				},
				HostedZoneId: &hostzone.Id,
			}
			if len(change) != 0 {
				log.Printf("change resource record sets input: %s", *input.HostedZoneId)
				dnsdeletememberchan <- &input
			}
			mutex.Unlock()

		}
		close(ipchan)
		close(dnsdeletememberchan)
	}()

}

func deleteDNSRecord(ctx context.Context, client *route53.Client) {
	log.Println("deleteDNSRecord running")
	input := <-dnsdeletememberchan
	output, err := client.ChangeResourceRecordSets(ctx, input)
	if err != nil {
		log.Printf("Delete Resource Record Set Error: %s", err.Error())

	} else {
		log.Printf("Successfully deleted : %s", output.ChangeInfo.Status)
	}
}

func cleanup(ctx context.Context, event events.CloudWatchEvent) {

	terminationdetail := TerminationDetail{}
	err := json.Unmarshal(event.Detail, &terminationdetail)
	if err != nil {
		log.Printf("Unmarshal Error: %s", err.Error())
	} else {
		log.Printf("Event: %s", terminationdetail)
	}
	var host Hostzone
	host.Id = hostzone.Id
	host.Hostname = terminationdetail.EC2InstanceId + "." + hostzone.Hostname

	go func() {
		hostchan <- host
	}()

	r53client := route53.NewFromConfig(cfg)
	eipclient := ec2.NewFromConfig(cfg)
	go func() {
		for {
			time.Sleep(10 * time.Microsecond)

			listDNSRecord(ctx, r53client, host)
		}
	}()

	go func() {
		for {
			time.Sleep(10 * time.Microsecond)

			prepareDNSRecord(ctx, host)
		}
	}()

	go func() {
		for {
			time.Sleep(10 * time.Microsecond)

			deleteDNSRecord(ctx, r53client)
		}
	}()

	go func() {
		for {
			time.Sleep(10 * time.Microsecond)

			listEIP(ctx, eipclient)
		}
	}()

	go func() {
		for {
			time.Sleep(10 * time.Microsecond)

			releaseEIP(ctx, eipclient)
		}
	}()

	time.Sleep(10 * time.Second)
}

func init() {
	var cfgerr error
	hostchan = make(chan Hostzone)
	dnsreousrcelistchan = make(chan *route53.ListResourceRecordSetsOutput)
	dnsdeletememberchan = make(chan *route53.ChangeResourceRecordSetsInput)
	ipchan = make(chan string)
	eiplistchan = make(chan *ec2.DescribeAddressesOutput)
	cfg, cfgerr = config.LoadDefaultConfig(context.TODO(), config.WithRegion("ap-northeast-2"))
	if cfgerr != nil {
		log.Fatalf("failed to load configuration, %v", cfgerr)
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
