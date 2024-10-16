package validate

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"path/filepath"

	v1 "gke-internal.googlesource.com/k8ssecurityvalidation_pa/client.git/v1"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/engine"
)

type HelmFetcher struct {
	RootDir string
}

func (h *HelmFetcher) PrepareValidateRequestContent(ctx context.Context) ([]RequestObject, error) {
	var result []RequestObject
	err := filepath.Walk(h.RootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// Check if this directory contains a Chart.yaml file
		if info.IsDir() {
			fmt.Printf("checking %s\n", path)
			chartPath := filepath.Join(path, "Chart.yaml")
			if _, err := os.Stat(chartPath); err == nil {
				fmt.Printf("Found chart in: %s\n", path)
				maps, err := renderYamlOffline(path)
				if err == nil {
					for name, value := range maps {
						result = append(result, RequestObject{
							ResourceName: name,
							Content: &v1.Content{
								ContentType: "1",
								Data:        base64.StdEncoding.EncodeToString([]byte(value)),
							},
						})
					}
				}
			}
		}
		return nil
	})
	return result, err
}

// renderYamlOffline generates and prints the YAML from a Helm chart without needing a cluster
func renderYamlOffline(chartPath string) (map[string]string, error) {
	// Load the chart
	chart, err := loader.Load(chartPath)
	if err != nil {
		log.Printf("Failed to load chart from %s: %v", chartPath, err)
		return nil, fmt.Errorf("Failed to load chart from %s: %v", chartPath, err)
	}

	// Create a render values (typically would come from values.yaml or --set)
	values := chart.Values // Load default values from the chart

	// Combine chart and default values
	options := chartutil.ReleaseOptions{
		Name:      "dry-run-release",
		Namespace: "default",
	}
	caps := chartutil.DefaultCapabilities

	// Create a config object to pass to the renderer
	config, err := chartutil.ToRenderValues(chart, values, options, caps)
	if err != nil {
		log.Printf("Failed to create render values for chart %s: %v", chartPath, err)
		return nil, fmt.Errorf("Failed to create render values for chart %s: %v", chartPath, err)
	}

	// Render the templates
	rendered, err := engine.Render(chart, config)
	if err != nil {
		log.Printf("Failed to render chart %s: %v", chartPath, err)
		return nil, fmt.Errorf("Failed to render chart %s: %v", chartPath, err)
	}

	result := map[string]string{}
	// Print the resulting YAML
	for name, content := range rendered {
		// fmt.Printf("YAML for template %s in chart %s:\n%s\n", name, chartPath, content)
		result[fmt.Sprintf("template %s in chart %s", name, chartPath)] = content
	}
	return result, nil
}
