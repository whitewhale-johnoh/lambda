package main

// Use this code snippet in your app.
// If you need more information about configurations or implementing the sample code, visit the AWS docs:
// https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/setting-up.html

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	runtime "github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
)

type Credential struct {
	AccessKey string
	SecretKey string
}

func getSecret() {
	secretName := "lambda/credential/config/"
	region := "ap-northeast-2"

	//Create a Secrets Manager client
	ccfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
	)
	if err != nil {
		// handle error
		fmt.Println(err.Error())
	}

	client := secretsmanager.NewFromConfig(ccfg)

	input := &secretsmanager.GetSecretValueInput{
		SecretId:     aws.String(secretName),
		VersionStage: aws.String("AWSCURRENT"), // VersionStage defaults to AWSCURRENT if unspecified
	}

	// In this sample we only handle the specific exceptions for the 'GetSecretValue' API.
	// See https://docs.aws.amazon.com/secretsmanager/latest/apireference/API_GetSecretValue.html

	result, err := client.GetSecretValue(context.TODO(), input)
	if err != nil {
		switch err.(type) {
		case *types.DecryptionFailure:
			// Secrets Manager can't decrypt the protected secret text using the provided KMS key.
			fmt.Println(err.Error())

		case *types.InternalServiceError:
			// An error occurred on the server side.
			fmt.Println(err.Error())

		case *types.InvalidParameterException:
			// You provided an invalid value for a parameter.
			fmt.Println(err.Error())

		case *types.InvalidRequestException:
			// You provided a parameter value that is not valid for the current state of the resource.
			fmt.Println(err.Error())

		case *types.ResourceNotFoundException:
			// We can't find the resource that you asked for.
			fmt.Println(err.Error())
		default:

			fmt.Println(err.Error())
		}

		return
	}

	// Decrypts secret using the associated KMS CMK.
	// Depending on whether the secret is a string or binary, one of these fields will be populated.
	var secretString, decodedBinarySecret string
	fmt.Println("String")
	fmt.Println(*result.SecretString)
	fmt.Println("binary")
	fmt.Println(string(result.SecretBinary))
	if result.SecretString != nil {
		secretString = *result.SecretString
	} else {
		decodedBinarySecretBytes := make([]byte, base64.StdEncoding.DecodedLen(len(result.SecretBinary)))
		len, err := base64.StdEncoding.Decode(decodedBinarySecretBytes, result.SecretBinary)
		if err != nil {
			fmt.Println("Base64 Decode Error:", err)
			return
		}
		decodedBinarySecret = string(decodedBinarySecretBytes[:len])
	}

	var cred Credential
	temp := stringToBin(secretString)
	json.Unmarshal(temp, &cred)
	fmt.Println(cred)
	// Your code goes here.
	fmt.Println("String2")
	fmt.Println(secretString)
	fmt.Println("binary2")
	fmt.Println(decodedBinarySecret)
}

// ...

func init() {

}

func main() {
	runtime.Start(getSecret)
}

func stringToBin(s string) []byte {
	var binString []byte
	for _, c := range s {
		binString = fmt.Sprintf("%s%b", binString, c)
	}
	return binString
}
