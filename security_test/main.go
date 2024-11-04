package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/blackzlq/k8s-validation-service-integration-demo/validate"
	"google.golang.org/api/container/v1"
)

func main() {
	ctx := context.Background()
	var fetcher validate.Fetcher
	mode := os.Getenv("MODE")

	switch mode {
	case "Helm":
		fetcher = &validate.HelmFetcher{
			RootDir: "./",
		}
	case "Static":
		fetcher = &validate.StaticFetcher{
			RootDir: "./",
		}
	default:
		location := os.Getenv("LOCATION")
		if location == "" {
			log.Fatalf("LOCATION environment variable not set")
		}

		clusterName := os.Getenv("CLUSTER_NAME")
		if clusterName == "" {
			log.Fatalf("CLUSTER_NAME environment variable not set")
		}

		focusComponentsPath := os.Getenv("FOCUS_COMPONENT_PATH")
		if focusComponentsPath == "" {
			log.Printf("FOCUS_COMPONENT_PATH is not set, will scan all component")
		}
		fetcher = &validate.ClusterFetcher{
			FocusComponentConfigPath: focusComponentsPath,
			CreateClusterRequest:     createClusterRequest,
			Location:                 location,
			ClusterName:              clusterName,
		}
	}

	resultHandler := &validate.DefaultResultHandler{
		ShouldReportViolation: validate.ShouldReport,
		PurifyViolation:       validate.PurifyViolation,
	}
	validator, err := validate.NewValidator(ctx, fetcher, resultHandler)
	if err != nil {
		log.Fatalf("failed to create validator")
	}
	if err = validator.Validate(ctx); err != nil {
		log.Fatalf("Validation failed: %v", err)
	}

	log.Println("Validation completed successfully")
}

func createClusterRequest() *container.CreateClusterRequest {
	location := os.Getenv("LOCATION")
	if location == "" {
		log.Fatalf("LOCATION environment variable not set")
	}

	clusterName := os.Getenv("CLUSTER_NAME")
	if clusterName == "" {
		log.Fatalf("CLUSTER_NAME environment variable not set")
	}

	projectID, err := validate.GetProjectID()
	if err != nil {
		log.Fatalf("Failed to get projectID")
	}
	parent := fmt.Sprintf("projects/%s/locations/%s", projectID, location)

	return &container.CreateClusterRequest{
		ProjectId: projectID,
		Parent:    parent,
		Cluster: &container.Cluster{
			Name:             clusterName,
			InitialNodeCount: 2, // Example: start with 2 nodes
			NodeConfig: &container.NodeConfig{
				MachineType: "e2-medium", // Example: machine type
				DiskSizeGb:  100,         // Example: disk size
			},
		},
	}
}
