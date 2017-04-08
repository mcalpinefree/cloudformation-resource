package main

import (
	"encoding/json"
	"fmt"

	"github.com/ci-pipeline/cloudformation-resource/utils"
)

func main() {
	result := utils.Result{Metadata: []string{}}
	output, _ := json.Marshal(result)
	fmt.Println(string(output))
}
