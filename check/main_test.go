package main

import (
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/ci-pipeline/cloudformation-resource/utils"
)

func TestGetVersionsNoStack(t *testing.T) {
	var input utils.Input
	var resp *cloudformation.DescribeStacksOutput
	var errArg error

	input = utils.Input{}
	resp = &cloudformation.DescribeStacksOutput{}
	errArg = errors.New("Stack does not exist")

	result := getVersions(input, resp, errArg)

	if len(result) > 0 {
		t.Error("Versions was %v but should have been an empty slice")
	}
}

func TestGetVersionsNoPrevious(t *testing.T) {
	var input utils.Input
	var resp *cloudformation.DescribeStacksOutput
	var errArg error

	input = utils.Input{}

	stacks := []*cloudformation.Stack{&cloudformation.Stack{}}
	resp = &cloudformation.DescribeStacksOutput{Stacks: stacks}

	result := getVersions(input, resp, errArg)

	if len(result) != 1 {
		t.Errorf("Expected len(result) == 1 but was %d: %v", len(result), result)
	}

	if result[0].LastUpdatedTime != "nil" {
		t.Errorf("Expected result[0].LastUpdatedTime == \"nil\", but was %v", result[0].LastUpdatedTime)
	}
}

func TestGetVersionsWithPreviousTheSame(t *testing.T) {
	var input utils.Input
	var resp *cloudformation.DescribeStacksOutput
	var errArg error

	input = utils.Input{}

	now := time.Now()
	input.Version.LastUpdatedTime = now.String()
	stacks := []*cloudformation.Stack{&cloudformation.Stack{LastUpdatedTime: &now}}
	resp = &cloudformation.DescribeStacksOutput{Stacks: stacks}

	result := getVersions(input, resp, errArg)

	if len(result) != 1 {
		t.Errorf("Expected len(result) == 1 but was %d: %v", len(result), result)
	}

	if result[0].LastUpdatedTime != now.String() {
		t.Errorf("Expected result[0].LastUpdatedTime == \"%s\", but was %v", now.String(), result[0].LastUpdatedTime)
	}
}

func TestGetVersionsWithPreviousDifferent(t *testing.T) {
	var input utils.Input
	var resp *cloudformation.DescribeStacksOutput
	var errArg error

	input = utils.Input{}
	input.Version.LastUpdatedTime = time.Now().String()
	time.Sleep(5 * time.Millisecond)

	now := time.Now()
	stacks := []*cloudformation.Stack{&cloudformation.Stack{LastUpdatedTime: &now}}
	resp = &cloudformation.DescribeStacksOutput{Stacks: stacks}

	result := getVersions(input, resp, errArg)

	if len(result) != 2 {
		t.Errorf("Versions was not length 2 but length %d: %v", len(result), result)
	}

	if result[0].LastUpdatedTime != input.Version.LastUpdatedTime {
		t.Errorf("Expected result[0].LastUpdatedTime == \"\", but was %v", result[0].LastUpdatedTime)
	}

	if result[1].LastUpdatedTime != now.String() {
		t.Errorf("Expected result[1].LastUpdatedTime == \"%s\", but was %v", now.String(), result[1].LastUpdatedTime)
	}
}
