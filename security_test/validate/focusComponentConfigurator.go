package validate

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"sync"

	v1 "gke-internal.googlesource.com/k8ssecurityvalidation_pa/client.git/v1"
)

// Cache to store precompiled patterns and focusComponentMap
var (
	focusComponentMap               map[string]*regexp.Regexp
	focusComponentMapCacheLoadOnce  sync.Once
	focusComponentMapCacheLoadError error
)

// loadFocusComponentMap loads and caches the focus component map
func loadFocusComponentMap(filePath string) error {
	if filePath == "" {
		focusComponentMap = make(map[string]*regexp.Regexp)
		return nil
	}

	// Read JSON file content
	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Temporary map to parse JSON and compile regex patterns
	var rawMap map[string]string
	if err := json.Unmarshal(fileContent, &rawMap); err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	// Precompile regex patterns and store them in focusComponentMap
	focusComponentMap = make(map[string]*regexp.Regexp, len(rawMap))
	for categoryName, pattern := range rawMap {
		reg, err := regexp.Compile(pattern)
		if err != nil {
			log.Printf("Error compiling regex for %s: %v, skipping", categoryName, err)
			continue
		}
		focusComponentMap[categoryName] = reg
	}

	return nil
}

// FindFocusComponent attempts to find the component's category based on name
func FindFocusComponent(name, filePath string) (string, bool) {
	// Load the focusComponentMap cache once
	focusComponentMapCacheLoadOnce.Do(func() {
		focusComponentMapCacheLoadError = loadFocusComponentMap(filePath)
	})

	// Handle load error (if any)
	if focusComponentMapCacheLoadError != nil {
		log.Fatalf("Failed to load focus component map: %v", focusComponentMapCacheLoadError)
	}

	// If no file path was provided, indicate a default match without a specific category
	if filePath == "" {
		return "", true
	}

	// Search for matching category in precompiled patterns
	for categoryName, reg := range focusComponentMap {
		if reg.MatchString(name) {
			return categoryName, true
		}
	}

	return "", false
}

func PurifyViolation(v *v1.Violation) *v1.Violation {
	return v
}

func ShouldReportBasedOnAllowList(v *v1.Violation, allowedResourceKeysMap map[string][]*v1.Violation) bool {
	_, exists := allowedResourceKeysMap[FetchResourceKey(v)]
	return !exists
}
