package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/ci-pipeline/cloudformation-resource/utils"
	"github.com/concourse/atc"
)

func main() {
	utils.Logln("Change to build directory")
	utils.GoToBuildDirectory()
	cwd, _ := os.Getwd()
	utils.Logln(cwd)
	input := utils.GetInput()
	svc := utils.GetCloudformationService(input)
	metadata, success := out(input, svc)
	result := utils.VersionResult{Metadata: metadata}
	output, _ := json.Marshal(result)
	fmt.Printf("%s", string(output))
	if !success {
		os.Exit(1)
	}
}

type Parameter struct {
	ParameterValue   string
	ParameterKey     string
	UsePreviousValue bool
}

type Tag struct {
	TagKey   string
	TagValue string
}

func stackExists(svc *cloudformation.CloudFormation, stackName string) bool {
	params := &cloudformation.DescribeStacksInput{
		StackName: aws.String(stackName),
	}
	req, resp := svc.DescribeStacksRequest(params)
	err := utils.HandleRequest(req)
	if err != nil {
		return false
	}
	utils.Logln(resp)

	return resp.Stacks[0] != nil
}

func waitForStack(svc *cloudformation.CloudFormation, input utils.Input) (success bool, arn, timestamp string) {
	success = true
	params := &cloudformation.DescribeStackEventsInput{
		StackName: aws.String(input.Source.Name),
	}

	pos := 0
	var stackEvents []*cloudformation.StackEvent
	for {
		req, resp := svc.DescribeStackEventsRequest(params)
		err := utils.HandleRequest(req)
		if err != nil {
			if input.Params.Delete {
				success = true
				utils.Logln("Stack deleted")
				return
			}
			utils.Logf("An error occured: %v\n", err)
			return
		}
		stackEvents = resp.StackEvents
		for j := len(stackEvents) - 1 - pos; j > -1; j-- {
			utils.Logln(stackEvents[j])
		}
		pos = len(stackEvents)

		arn = *stackEvents[0].StackId
		timestamp = stackEvents[0].Timestamp.String()

		if *stackEvents[0].ResourceType == "AWS::CloudFormation::Stack" {
			if *stackEvents[0].ResourceStatus == "CREATE_COMPLETE" {
				return
			} else if *stackEvents[0].ResourceStatus == "UPDATE_COMPLETE" {
				return
			} else if *stackEvents[0].ResourceStatus == "ROLLBACK_COMPLETE" {
				success = false
				return
			} else if *stackEvents[0].ResourceStatus == "UPDATE_ROLLBACK_COMPLETE" {
				success = false
				return
			} else if strings.HasSuffix(*stackEvents[0].ResourceStatus, "_FAILED") {
				success = false
				return
			}
		}

		time.Sleep(500 * time.Millisecond)
	}
}

func out(input utils.Input, svc *cloudformation.CloudFormation) (metadata []atc.MetadataField, success bool) {

	var capabilities []*string
	capabilities = nil
	if input.Params.Capabilities != nil {
		for _, c := range input.Params.Capabilities {
			capabilities = append(capabilities, aws.String(c))
		}
	}

	var templateBody string
	if input.Params.Template != "" {
		bytes, err := ioutil.ReadFile(input.Params.Template)
		if err != nil {
			panic(err)
		}
		templateBody = string(bytes)
	}

	parameters := []Parameter{}
	var cloudformationParams []*cloudformation.Parameter
	if input.Params.Parameters != "" {
		bytes, err := ioutil.ReadFile(input.Params.Parameters)
		err = json.Unmarshal(bytes, &parameters)
		if err != nil {
			utils.Logln("Could not unmarshal parameters file, path = ", input.Params.Parameters)
			utils.Logln("contents:")
			utils.Logln(string(bytes))
			utils.Logln("error: ", err)
			os.Exit(1)
		}
	}
	for _, p := range parameters {
		param := cloudformation.Parameter{
			ParameterKey:     aws.String(p.ParameterKey),
			ParameterValue:   aws.String(p.ParameterValue),
			UsePreviousValue: aws.Bool(p.UsePreviousValue),
		}
		cloudformationParams = append(cloudformationParams, &param)
	}

	tags := []Tag{}
	var cloudformationTags []*cloudformation.Tag
	if input.Params.Tags != "" {
		bytes, err := ioutil.ReadFile(input.Params.Tags)
		err = json.Unmarshal(bytes, &tags)
		if err != nil {
			panic(err)
		}
	}
	for _, t := range tags {
		tag := cloudformation.Tag{
			Key:   aws.String(t.TagKey),
			Value: aws.String(t.TagValue),
		}
		cloudformationTags = append(cloudformationTags, &tag)
	}

	if input.Params.Delete == true {
		utils.Logln("Deleting stack")
		params := &cloudformation.DeleteStackInput{
			StackName: aws.String(input.Source.Name),
		}
		req, resp := svc.DeleteStackRequest(params)
		err := utils.HandleRequest(req)
		if err != nil {
			utils.Logln(err.Error())
		}
		utils.Logln(resp)
	} else if !stackExists(svc, input.Source.Name) {
		utils.Logln("Creating stack")
		params := &cloudformation.CreateStackInput{
			StackName:    aws.String(input.Source.Name),
			Capabilities: capabilities,
			Parameters:   cloudformationParams,
			Tags:         cloudformationTags,
			TemplateBody: aws.String(templateBody),
		}
		req, resp := svc.CreateStackRequest(params)
		err := utils.HandleRequest(req)
		if err != nil {
			utils.Logln(err.Error())
		}
		utils.Logln(resp)
	} else {
		utils.Logln("Updating stack")
		params := &cloudformation.UpdateStackInput{
			StackName:    aws.String(input.Source.Name),
			Capabilities: capabilities,
			Parameters:   cloudformationParams,
			Tags:         cloudformationTags,
			TemplateBody: aws.String(templateBody),
		}
		req, _ := svc.UpdateStackRequest(params)
		err := utils.HandleRequest(req)
		if err != nil {
			if awsErr, ok := err.(awserr.Error); ok {
				if awsErr.Code() == "ValidationError" && awsErr.Message() == "No updates are to be performed." {
					utils.Logln("No updates to be performed")
				} else {
					utils.Logln("An AWS error occured whilst updating stack: ", err)
					os.Exit(1)
				}
			} else {
				utils.Logln("An error occured whilst updating stack: ", err)
				os.Exit(1)
			}
		}
	}

	success, arn, timestamp := waitForStack(svc, input)
	result := make([]atc.MetadataField, 2)
	result[0].Name = "arn"
	result[0].Value = arn
	result[1].Name = "timestamp"
	result[1].Value = timestamp
	return result, success
}
