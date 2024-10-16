package validate

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"regexp"
	"strings"

	v1 "gke-internal.googlesource.com/k8ssecurityvalidation_pa/client.git/v1"
)

func FindFocusComponent(name string, focusComponentMap map[string]string) (string, bool) {
	for categoryName, pattern := range focusComponentMap {
		matched, err := regexp.MatchString(pattern, name)
		if err != nil {
			fmt.Printf("Error compiling regex: %v\n", err)
			continue
		}
		if matched {
			return categoryName, true
		}
	}
	return "", false
}

func AllowedResourceKeysMap(allowList embed.FS) map[string][]*v1.Violation {
	violations, err := allowedViolations(allowList)
	keyMap := map[string][]*v1.Violation{}
	if err != nil {
		fmt.Println("failed to load allowlist")
		return keyMap
	}
	for _, violation := range violations {
		key := FetchResourceKey(violation)
		keyMap[key] = append(keyMap[key], violation)
	}
	return keyMap
}

func allowedViolations(allowList embed.FS) ([]*v1.Violation, error) {
	var vs []*v1.Violation
	// Walk through all embedded files in the "folder" directory
	err := fs.WalkDir(allowList, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// Check if the file has a .json extension
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".json") {
			violations, err := readViolationsFromFile(allowList, path)
			if err != nil {
				return err
			}
			vs = append(vs, violations...)
		}
		return nil
	})
	if err != nil {
		fmt.Println("Error:", err)
	}
	return vs, nil
}

func readViolationsFromFile(fs embed.FS, name string) ([]*v1.Violation, error) {
	violations := make([]*v1.Violation, 0)
	file, _ := fs.Open(name)
	defer file.Close()
	data, _ := io.ReadAll(file)
	if err := json.Unmarshal(data, &violations); err != nil {
		return nil, err
	}
	return violations, nil
}

func PurifyViolation(v *v1.Violation) *v1.Violation {
	return v
}

func ShouldReport(v *v1.Violation, allowedResourceKeysMap map[string][]*v1.Violation) bool {
	_, exists := allowedResourceKeysMap[FetchResourceKey(v)]
	return !exists
}
