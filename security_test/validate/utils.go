package validate

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	v1 "gke-internal.googlesource.com/k8ssecurityvalidation_pa/client.git/v1"
)

func GetProjectID() (string, error) {
	cmd := exec.Command("gcloud", "config", "get", "project")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out.String()), nil
}

func getAccessToken() (string, error) {
	cmd := exec.Command("gcloud", "auth", "print-access-token")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out.String()), nil
}

func executeCommand(command string) (string, error) {
	cmd := exec.Command("bash", "-c", command)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", err
	}
	return out.String(), nil
}

func FetchResourceKey(violation *v1.Violation) string {
	if violation == nil || violation.ResourceKey == nil {
		return ""
	}
	return fmt.Sprintf("policyName: %s, group: %s, kind: %s, name: %s, version: %s",
		violation.PolicyName, violation.ResourceKey.Group, violation.ResourceKey.Kind,
		violation.ResourceKey.Name, violation.ResourceKey.Version)
}
