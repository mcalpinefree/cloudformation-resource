package main

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"errors"
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
	metadata, success, version := out(input, svc)
	result := utils.VersionResult{Metadata: metadata, Version: atc.Version{"sha1": version}}
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

type ApiInputs struct {
	Capabilities         []*string
	CloudformationParams []*cloudformation.Parameter
	TemplateBody         string
	CloudformationTags   []*cloudformation.Tag
}

func stackExists(svc *cloudformation.CloudFormation, stackName string) bool {
	params := &cloudformation.DescribeStacksInput{
		StackName: aws.String(stackName),
	}
	resp, err := svc.DescribeStacks(params)
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
		resp, err := svc.DescribeStackEvents(params)
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

func createChangeSet(input utils.Input, svc *cloudformation.CloudFormation, apiInputs ApiInputs) error {
	var changeSetType string
	if stackExists(svc, input.Source.Name) {
		changeSetType = cloudformation.ChangeSetTypeUpdate
	} else {
		changeSetType = cloudformation.ChangeSetTypeCreate
	}
	resp, err := svc.CreateChangeSet(&cloudformation.CreateChangeSetInput{
		Capabilities:  apiInputs.Capabilities,
		ChangeSetName: aws.String("concourse-" + time.Now().Format("20060102150405")),
		ChangeSetType: aws.String(changeSetType),
		Description:   aws.String("Changeset created by concourse"),
		Parameters:    apiInputs.CloudformationParams,
		StackName:     aws.String(input.Source.Name),
		Tags:          apiInputs.CloudformationTags,
		TemplateBody:  aws.String(apiInputs.TemplateBody),
	})
	if err != nil {
		return err
	}
	utils.Logln(resp)
	return nil
}

func executeChangeSet(input utils.Input, svc *cloudformation.CloudFormation, apiInputs ApiInputs) error {
	changeSets, err := svc.ListChangeSets(&cloudformation.ListChangeSetsInput{
		StackName: aws.String(input.Source.Name),
	})
	if err != nil {
		return err
	}
	if len(changeSets.Summaries) != 1 {
		return errors.New("There must only be 1 changeset associated with a stack to execute")
	}
	resp, err := svc.ExecuteChangeSet(&cloudformation.ExecuteChangeSetInput{
		StackName:     aws.String(input.Source.Name),
		ChangeSetName: changeSets.Summaries[0].ChangeSetName,
	})
	if err != nil {
		return err
	}
	utils.Logln(resp)
	return nil
}

func createStack(input utils.Input, svc *cloudformation.CloudFormation, apiInputs ApiInputs) error {
	utils.Logln("Creating stack")
	params := &cloudformation.CreateStackInput{
		StackName:    aws.String(input.Source.Name),
		Capabilities: apiInputs.Capabilities,
		Parameters:   apiInputs.CloudformationParams,
		Tags:         apiInputs.CloudformationTags,
		TemplateBody: aws.String(apiInputs.TemplateBody),
	}
	resp, err := svc.CreateStack(params)
	if err != nil {
		utils.Logln(err.Error())
	}
	utils.Logln(resp)
	return nil
}

func updateStack(input utils.Input, svc *cloudformation.CloudFormation, apiInputs ApiInputs) error {
	utils.Logln("Updating stack")
	params := &cloudformation.UpdateStackInput{
		StackName:    aws.String(input.Source.Name),
		Capabilities: apiInputs.Capabilities,
		Parameters:   apiInputs.CloudformationParams,
		Tags:         apiInputs.CloudformationTags,
		TemplateBody: aws.String(apiInputs.TemplateBody),
	}
	_, err := svc.UpdateStack(params)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == "ValidationError" && awsErr.Message() == "No updates are to be performed." {
				utils.Logln("No updates to be performed")
				return nil
			} else {
				return errors.New(fmt.Sprintf("An AWS error occured whilst updating stack: %v", err))
			}
		} else {
			return errors.New(fmt.Sprintf("An error occured whilst updating stack: %v", err))
		}
	}
	return nil
}
func deleteStack(input utils.Input, svc *cloudformation.CloudFormation, apiInputs ApiInputs) error {
	utils.Logln("Deleting stack")
	resp, err := svc.DeleteStack(&cloudformation.DeleteStackInput{
		StackName: aws.String(input.Source.Name),
	})
	if err != nil {
		return err
	}
	utils.Logln(resp)
	return nil
}

func out(input utils.Input, svc *cloudformation.CloudFormation) (metadata []atc.MetadataField, success bool, sha string) {

	var apiInputs ApiInputs

	apiInputs.Capabilities = nil
	if input.Params.Capabilities != nil {
		for _, c := range input.Params.Capabilities {
			apiInputs.Capabilities = append(apiInputs.Capabilities, aws.String(c))
		}
	}

	if input.Params.Template != "" {
		bytes, err := ioutil.ReadFile(input.Params.Template)
		if err != nil {
			panic(err)
		}
		apiInputs.TemplateBody = string(bytes)
	}

	parameters := []Parameter{}
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
		apiInputs.CloudformationParams = append(apiInputs.CloudformationParams, &param)
	}

	tags := []Tag{}
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
		apiInputs.CloudformationTags = append(apiInputs.CloudformationTags, &tag)
	}

	if input.Params.Delete == true {
		err := deleteStack(input, svc, apiInputs)
		if err != nil {
			utils.Logln("Could not delete stack: ", err)
		}
	} else if input.ChangesetCreate() || input.ChangesetExecute() {
		if input.ChangesetCreate() {
			err := createChangeSet(input, svc, apiInputs)
			if err != nil {
				utils.Logln("Could not create Change Set: ", err)
			}
		}
		if input.ChangesetExecute() {
			err := executeChangeSet(input, svc, apiInputs)
			if err != nil {
				utils.Logln("Could not execute Change Set: ", err)
			}
		}
	} else {
		if !stackExists(svc, input.Source.Name) {
			err := createStack(input, svc, apiInputs)
			if err != nil {
				utils.Logln("Could not create stack: ", err)
			}
		} else {
			err := updateStack(input, svc, apiInputs)
			if err != nil {
				utils.Logln("Could not update stack: ", err)
			}
		}
	}

	success, arn, timestamp := waitForStack(svc, input)
	result := make([]atc.MetadataField, 2)
	result[0].Name = "arn"
	result[0].Value = arn
	result[1].Name = "timestamp"
	result[1].Value = timestamp

	// SHA1 template, parameters and tags and use as version
	hasher := sha1.New()
	hasher.Write([]byte(fmt.Sprintf("%s%v%v", apiInputs.TemplateBody, parameters, tags)))
	sha = base64.URLEncoding.EncodeToString(hasher.Sum(nil))
	return result, success, sha
}
