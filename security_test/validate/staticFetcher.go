package validate

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	v1 "gke-internal.googlesource.com/k8ssecurityvalidation_pa/client.git/v1"
)

type StaticFetcher struct {
	RootDir string
}

func (s *StaticFetcher) PrepareValidateRequestContent(ctx context.Context) ([]RequestObject, error) {
	var result []RequestObject

	// Step 1: Run Terraform plan
	planOutput, err := s.runTerraformPlan()
	if err != nil {
		return nil, fmt.Errorf("failed to run terraform plan: %v", err)
	}

	// Step 2: Parse the Terraform plan output
	yamlFiles, err := s.extractYamlFromPlan(planOutput)
	if err != nil {
		return nil, fmt.Errorf("failed to extract YAML from plan: %v", err)
	}

	// Step 3: Read and encode YAML files
	for _, yamlPath := range yamlFiles {
		content, err := os.ReadFile(yamlPath)
		if err == nil {
			result = append(result, RequestObject{
				ResourceName: filepath.Base(yamlPath),
				Content: &v1.Content{
					ContentType: "1",
					Data:        base64.StdEncoding.EncodeToString(content),
				},
			})
		} else {
			log.Printf("Failed to read YAML file %s: %v\n", yamlPath, err)
		}
	}

	return result, nil
}

// runTerraformPlan runs terraform plan and captures the output in JSON format
func (s *StaticFetcher) runTerraformPlan() ([]byte, error) {
	cmd := exec.Command("terraform", "plan", "-out=plan.out", "-input=false", "-no-color")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("terraform plan failed: %v\nOutput: %s", err, out.String())
	}

	// Convert the binary plan output to JSON
	showCmd := exec.Command("terraform", "show", "-json", "plan.out")
	var showOut bytes.Buffer
	showCmd.Stdout = &showOut
	showCmd.Stderr = &showOut

	err = showCmd.Run()
	if err != nil {
		return nil, fmt.Errorf("terraform show failed: %v\nOutput: %s", err, showOut.String())
	}

	return showOut.Bytes(), nil
}

// extractYamlFromPlan parses the Terraform plan JSON output to extract YAML file paths
func (s *StaticFetcher) extractYamlFromPlan(planOutput []byte) ([]string, error) {
	var yamlFiles []string

	var planData map[string]interface{}
	if err := json.Unmarshal(planOutput, &planData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal plan output: %v", err)
	}

	// Traverse the plan data to find any resources that might generate or reference YAML files
	if resources, ok := planData["planned_values"].(map[string]interface{})["root_module"].(map[string]interface{})["resources"].([]interface{}); ok {
		for _, resource := range resources {
			if res, ok := resource.(map[string]interface{}); ok {
				attributes := res["values"].(map[string]interface{})
				for key, value := range attributes {
					if strings.HasSuffix(key, "_yaml_file") {
						if yamlFilePath, ok := value.(string); ok && yamlFilePath != "" {
							yamlFiles = append(yamlFiles, yamlFilePath)
						}
					}
				}
			}
		}
	}

	return yamlFiles, nil
}
