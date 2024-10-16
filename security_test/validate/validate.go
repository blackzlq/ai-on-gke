package validate

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
)

// Validator struct
type Validator struct {
	fetcher          Fetcher
	resultHandler    ResultHandler
	validationClient *ValidationClient
}

// NewValidator constructor
func NewValidator(ctx context.Context, fetcher Fetcher, resultHandler ResultHandler) (*Validator, error) {
	server, err := NewValidationServiceClient(ctx)
	if err != nil {
		return nil, err
	}
	return &Validator{
		fetcher:          fetcher,
		resultHandler:    resultHandler,
		validationClient: server,
	}, nil
}

// Validate method that orchestrates the validation process
func (v *Validator) Validate(ctx context.Context) error {
	// Fetch YAML files
	requestObjects, err := v.fetcher.PrepareValidateRequestContent(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch YAML files: %v", err)
	}

	// Call the client (stub for demonstration)
	resultObjects, err := v.scanViolation(ctx, requestObjects)
	if err != nil {
		return fmt.Errorf("failed to call client: %v", err)
	}

	// Handle the result
	err = v.resultHandler.HandleResult(ctx, resultObjects)
	if err != nil {
		return fmt.Errorf("failed to handle result: %v", err)
	}

	return nil
}

func (v *Validator) scanViolation(ctx context.Context, objects []RequestObject) ([]ResultObject, error) {
	queue := make([]string, 0)
	operationResourceMap := map[string]string{}
	for _, object := range objects {
		operation, err := v.validationClient.ValidateResource(v.validationClient.CreateValidateRequest(object.Content))
		if err != nil {
			return nil, fmt.Errorf("failed to call validate service, please contact Kubernetes hardening team (https://oncall.corp.google.com/cloud-kubernetes-hardening); error: %w", err)
		}
		queue = append(queue, operation.Name)
		operationResourceMap[operation.Name] = object.ResourceName
	}
	if len(queue) == 0 {
		fmt.Printf("no objects to validate")
		return nil, nil
	}
	var resultObjects []ResultObject
	err := wait.PollUntilContextTimeout(ctx, 1*time.Second, 120*time.Minute, true, func(context.Context) (bool, error) {
		name := queue[0]
		queue = queue[1:]
		operation, err := v.validationClient.RetrieveOperation(name)
		if err != nil {
			fmt.Printf("failed to get operation %v\n, will retry", name)
			queue = append(queue, name)
		}
		if operation == nil {
			return false, nil
		}
		if operation.Done {
			if operation.Error != nil {
				fmt.Printf("error when try to finish the operation, error: %v\n", operation.Error)
			} else {
				violations, err := v.validationClient.RetrieveViolationsFromOperation(operation)
				if err != nil {
					fmt.Printf("failed to parse response to violation, error: %v\n", err)
					fmt.Printf("yaml file: %s.yaml is not valid\n", operationResourceMap[operation.Name])
				}
				if v != nil {
					for _, violation := range violations {
						violation.ResourceKey.Name = operationResourceMap[operation.Name]
					}
					resultObjects = append(resultObjects, ResultObject{
						ResourceName: operationResourceMap[operation.Name],
						Violations:   violations,
					})
					fmt.Printf("yaml file: %s.yaml is valid\n", operationResourceMap[operation.Name])
				}
			}
		} else {
			queue = append(queue, name)
		}
		return len(queue) == 0, nil
	})
	if len(queue) > 0 {
		fmt.Printf("retried 3 times still can not empty the queue")
	}
	return resultObjects, err
}
