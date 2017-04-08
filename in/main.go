package main

import (
	"encoding/json"
	"fmt"

	"github.com/ci-pipeline/cloudformation-resource/utils"
)

func main() {
	metadata := make([]interface{}, 0)
	result := utils.Result{Metadata: metadata}
	output, _ := json.Marshal(result)
	fmt.Println(string(output))
}
