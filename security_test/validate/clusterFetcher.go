package validate

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
	v1 "gke-internal.googlesource.com/k8ssecurityvalidation_pa/client.git/v1"
	"google.golang.org/api/container/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
)

// 32 MB https://cloud.google.com/api-gateway/docs/quotas#payload_size_limits
const requestSizeLimit = 32 * 1024 * 1024

type CreateClusterRequestFunc func() *container.CreateClusterRequest

type ClusterFetcher struct {
	FocusComponentConfigPath string

	CreateClusterRequest CreateClusterRequestFunc
	Location             string
	ClusterName          string
}

func (c *ClusterFetcher) PrepareValidateRequestContent(ctx context.Context) ([]RequestObject, error) {
	projectID, err := GetProjectID()
	if err != nil {
		return nil, err
	}

	parent := fmt.Sprintf("projects/%s/locations/%s", projectID, c.Location)

	createCluster := os.Getenv("CREATE_CLUSTER")
	if createCluster == "" {
		createCluster = "true"
	}

	// Step 1: Create the cluster (stub for demonstration)
	if strings.ToLower(createCluster) != "false" {
		err = c.prepareCluster(ctx, projectID, parent, c.ClusterName, c.Location)
		if err != nil {
			return nil, fmt.Errorf("failed to create cluster: %v", err)
		}
	}

	// Step 2: Setup Kube Access
	if err := c.setupKubeAccess(projectID, c.Location, c.ClusterName); err != nil {
		log.Printf("Error setting kubectl credentials: %v", err)
	}

	// Step 3: Snapshot the cluster to get targeted YAML files (stub for demonstration)
	objects, err := c.prepareValidateRequestContent(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to snapshot cluster: %v", err)
	}

	// Step 4: Delete the cluster
	if strings.ToLower(createCluster) != "false" {
		err = c.cleanup(ctx, projectID, parent, c.ClusterName, c.Location)
		if err != nil {
			return nil, fmt.Errorf("failed to delete the cluster: %v", err)
		}
	}
	return objects, nil
}

func (c *ClusterFetcher) prepareCluster(ctx context.Context, projectID, parent, clusterName, location string) error {
	// 1. Create a Container Service
	svc, err := container.NewService(ctx)
	if err != nil {
		log.Fatalf("Error creating Container Service: %v", err)
		return err
	}
	// Placeholder for actual cluster creation
	fmt.Println("Creating cluster...")
	cluster, err := c.createGKECluster(svc, projectID, parent, clusterName, location)
	if err != nil {
		return err
	}
	fmt.Println("Cluster Name:", cluster.Name)
	fmt.Println("Status:", cluster.Status)
	fmt.Println("Endpoint:", cluster.Endpoint)
	fmt.Println("Node Pools:", cluster.NodePools)

	return waitClusterReachStatus(svc, parent, clusterName, "RUNNING")
}

func (c *ClusterFetcher) setupKubeAccess(projectID, location, clusterName string) error {
	cmd := exec.Command("gcloud", "container", "clusters", "get-credentials", clusterName, "--location", location, "--project", projectID)
	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = 5 * time.Minute
	checkOperationStatus := func() error {
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out

		return cmd.Run()
	}

	return backoff.Retry(checkOperationStatus, b)
}

func (c *ClusterFetcher) prepareValidateRequestContent(ctx context.Context) ([]RequestObject, error) {
	fmt.Println("Snapshotting cluster...")
	yamlMap, err := c.snapshotYAMLs()
	if err != nil {
		return nil, err
	}
	return convertYAMLStringToValidateRequestContent(yamlMap)
}

func (c *ClusterFetcher) cleanup(ctx context.Context, projectID, parent, clusterName, location string) error {
	fmt.Println("Deleting cluster...")
	return deleteGKECluster(ctx, projectID, parent, clusterName)
}

// createGKECluster creates a GKE cluster in the specified parent.
func (c *ClusterFetcher) createGKECluster(svc *container.Service, projectID, parent, clusterName, location string) (*container.Cluster, error) {
	// Define Your Cluster Configuration
	fmt.Printf("Creating Request...")
	// Send the Create Cluster Request
	op, err := svc.Projects.Locations.Clusters.Create(parent, c.CreateClusterRequest()).Do()
	if err != nil {
		log.Fatalf("Error creating cluster: %v", err)
		return nil, err
	}
	fmt.Printf("Create Cluster done")

	fmt.Printf("Cluster creation in progress: %s\n", op.Name)
	// projects/*/locations/*/clusters/*
	return svc.Projects.Locations.Clusters.Get(fmt.Sprintf("%s/clusters/%s", parent, clusterName)).Do()
}

func waitClusterReachStatus(svc *container.Service, parent, clusterName string, targetStatus string) error {
	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = 15 * time.Minute
	checkOperationStatus := func() error {
		cluster, err := svc.Projects.Locations.Clusters.Get(fmt.Sprintf("%s/clusters/%s", parent, clusterName)).Do()
		if err != nil {
			return fmt.Errorf("failed to get cluster: %w", err)
		}
		if cluster.Status == targetStatus {
			fmt.Printf("Cluster reach target status %s", targetStatus)
			return nil
		}
		fmt.Printf("Cluster current status: %s, want: %s\n", cluster.Status, targetStatus)
		return fmt.Errorf("cluster status is %s", cluster.Status)
	}
	return backoff.Retry(checkOperationStatus, b)
}

// deleteGKECluster deletes a GKE cluster in the specified parent.
func deleteGKECluster(ctx context.Context, projectID, parent, clusterName string) error {
	svc, err := container.NewService(ctx)
	if err != nil {
		return fmt.Errorf("error creating Container Service: %v", err)
	}

	// Send the Delete Cluster Request
	op, err := svc.Projects.Locations.Clusters.Delete(fmt.Sprintf("%s/clusters/%s", parent, clusterName)).Do()
	if err != nil {
		// Check for specific "cluster not found" error
		if strings.Contains(err.Error(), "notFound") {
			return fmt.Errorf("cluster %s not found in parent %s", clusterName, parent)
		}
		return fmt.Errorf("error deleting cluster: %v", err)
	}

	// Monitor the Deletion Operation (Optional)
	for {
		result, err := svc.Projects.Locations.Operations.Get(fmt.Sprintf("%s/operations/%s", parent, op.Name)).Do()
		if err != nil {
			return fmt.Errorf("error getting operation status: %v", err)
		}

		if result.Status == "DONE" {
			if result.Error != nil {
				return fmt.Errorf("cluster deletion failed: %v", result.Error)
			}
			fmt.Printf("Cluster %s successfully deleted from parent %s\n", clusterName, parent)
			return nil // Success!
		}
		time.Sleep(5 * time.Second) // Poll every 5 seconds
	}
}

func (c *ClusterFetcher) snapshotYAMLs() (map[string]string, error) {
	// 1. Load Kubeconfig (assumes you've set up kubectl access to your GKE cluster)
	kubeconfig := os.Getenv("HOME") + "/.kube/config" // Update if your kubeconfig is elsewhere
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		log.Fatalf("Error building kubeconfig: %v", err)
		return nil, err
	}

	// 2. Create Dynamic Client
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		log.Fatalf("Failed to create dynamic client: %v", err)
		return nil, err
	}

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		log.Fatalf("Failed to create discovery client: %v", err)
		return nil, err
	}

	yamlMap := map[string]string{}

	gvrs, err := getAllResources(discoveryClient)
	if err != nil {
		log.Fatalf("Failed to get all resources: %v", err)
		return nil, err
	}

	for _, gvr := range gvrs {
		fmt.Printf("resource is %v\n", gvr)
	}

	// Get all resources in all namespaces and print them in YAML format
	for _, gvr := range gvrs {
		resources, err := dynamicClient.Resource(getGVR(gvr)).Namespace("").List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			log.Printf("Failed to get resources: %v", err)
			continue
		}

		// Serialize to YAML
		for _, item := range resources.Items {
			name := item.GetName()
			if len(name) == 0 {
				name = "unknown"
			}
			categoryName, shouldScan := FindFocusComponent(name, c.FocusComponentConfigPath)
			if shouldScan {
				fmt.Printf("resource item name is %s\n", item.GetName())
				yamlData, err := runtimeToYAML(&item)
				if err != nil {
					log.Fatalf("Failed to serialize to YAML: %v", err)
					return nil, err
				}

				if len(yamlMap[categoryName]) != 0 {
					// YAML uses three dashes (“---”) to separate documents within a stream. https://yaml.org/spec/1.0/
					yamlMap[categoryName] += "\n---\n"
				}
				yamlMap[categoryName] += string(yamlData)
			}
		}
	}
	fmt.Println("finished scan resource to gather yaml files")
	return yamlMap, nil
}

func convertYAMLStringToValidateRequestContent(yamlMap map[string]string) ([]RequestObject, error) {
	var objects []RequestObject
	for name, yaml := range yamlMap {
		if base64.StdEncoding.EncodedLen(len(yaml)) >= requestSizeLimit {
			fmt.Printf("single YAML file size is over 32MB, google api do not support it, it is %v\n", base64.StdEncoding.EncodedLen(len(yaml)))
		} else {
			objects = append(objects, RequestObject{
				ResourceName: name,
				Content: &v1.Content{
					ContentType: "1",
					Data:        base64.StdEncoding.EncodeToString([]byte(yaml)),
				},
			})
		}
	}
	return objects, nil
}

func getAllResources(discoveryClient discovery.DiscoveryInterface) ([]schema.GroupVersionResource, error) {
	apiResourceLists, err := discoveryClient.ServerPreferredResources()
	if err != nil {
		return nil, fmt.Errorf("failed to get preferred resources: %v", err)
	}

	var gvrs []schema.GroupVersionResource
	for _, apiResourceList := range apiResourceLists {
		gv, err := schema.ParseGroupVersion(apiResourceList.GroupVersion)
		if err != nil {
			return nil, fmt.Errorf("failed to parse group version: %v", err)
		}

		for _, apiResource := range apiResourceList.APIResources {
			if !apiResource.Namespaced { // Skip non-namespaced resources for now
				continue
			}
			gvrs = append(gvrs, schema.GroupVersionResource{
				Group:    gv.Group,
				Version:  gv.Version,
				Resource: apiResource.Name,
			})
		}
	}
	return gvrs, nil

}

// getGVR maps a resource name to a GroupVersionResource
func getGVR(resource schema.GroupVersionResource) schema.GroupVersionResource {
	return resource
}

// runtimeToYAML serializes a runtime.Object to YAML
func runtimeToYAML(obj runtime.Object) ([]byte, error) {
	serializer := json.NewSerializerWithOptions(json.DefaultMetaFactory, scheme.Scheme, scheme.Scheme, json.SerializerOptions{Yaml: true, Pretty: true, Strict: false})
	yamlData, err := runtime.Encode(serializer, obj)
	if err != nil {
		return nil, err
	}
	return yamlData, nil
}
