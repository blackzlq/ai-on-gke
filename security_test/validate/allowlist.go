package validate

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"

	v1 "gke-internal.googlesource.com/k8ssecurityvalidation_pa/client.git/v1"
)

// Cache for allowed resources to avoid reloading
var (
	allowedResourceKeysMap     map[string][]*v1.Violation
	allowedResourceKeysMapOnce sync.Once
)

func loadAllowedResourceKeysMap(allowListFolder string) map[string][]*v1.Violation {
	// Load the allowed resources once and cache them
	allowedResourceKeysMapOnce.Do(func() {
		// Load violations from folder and handle any errors
		violations, err := allowedViolations(allowListFolder)
		if err != nil {
			fmt.Println("Failed to load allowlist:", err)
			allowedResourceKeysMap = map[string][]*v1.Violation{} // empty map on failure
			return
		}

		// Populate the keyMap with the violations
		allowedResourceKeysMap = map[string][]*v1.Violation{}
		for _, violation := range violations {
			key := FetchResourceKey(violation)
			allowedResourceKeysMap[key] = append(allowedResourceKeysMap[key], violation)
		}
	})

	// Return the cached map
	return allowedResourceKeysMap
}

// allowedViolations reads all JSON files in the specified folder and parses them into violations
func allowedViolations(allowListFolder string) ([]*v1.Violation, error) {
	var violations []*v1.Violation

	// Walk through the specified folder and process each file
	err := filepath.Walk(allowListFolder, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Only process JSON files
		if !info.IsDir() && filepath.Ext(path) == ".json" {
			fileContent, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("failed to read file %s: %w", path, err)
			}

			// Parse the JSON content
			var fileViolations []*v1.Violation
			if err := json.Unmarshal(fileContent, &fileViolations); err != nil {
				fmt.Printf("Failed to unmarshal JSON file %s: %v\n", path, err)
				return nil // Skip this file if unmarshalling fails
			}

			// Append violations from this file to the main list
			violations = append(violations, fileViolations...)
		}

		return nil
	})

	return violations, err
}

func ShouldReport(v *v1.Violation) bool {
	allowListFolder := os.Getenv("ALLOW_LIST_FOLDER")
	loadAllowedResourceKeysMap(allowListFolder)
	if allowedResourceKeysMap == nil || len(allowedResourceKeysMap) == 0 {
		// If the map is empty, report all violations
		return true
	}
	_, exists := allowedResourceKeysMap[FetchResourceKey(v)]
	return !exists
}
