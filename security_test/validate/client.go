package validate

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	v1 "gke-internal.googlesource.com/k8ssecurityvalidation_pa/client.git/v1"
)

// ValidationClient is a validation client.
type ValidationClient struct {
	Service   *v1.Service
	ProjectID string
}

// ViolationList is a list of violations.
type ViolationList struct {
	Violations []*v1.Violation `json:"violations"`
}

// NewValidationServiceClient creates a new API client used to call Google API service.
func NewValidationServiceClient(ctx context.Context) (*ValidationClient, error) {
	service, err := v1.NewService(ctx)

	if err != nil {
		return nil, err
	}
	return &ValidationClient{
		Service: service,
	}, nil
}

// createValidateRequest generates a validate request with the base64 encoded YAML string.
func (vc *ValidationClient) CreateValidateRequest(content *v1.Content) *v1.ValidateRequest {
	req := &v1.ValidateRequest{
		Resources: content,
	}
	return req
}

// ValidateResource makes up a validation request and performs an outbound request to retrieve the violation results. It will return a long-running operation.
func (vc *ValidationClient) ValidateResource(request *v1.ValidateRequest) (*v1.Operation, error) {
	vcall := vc.Service.V1.Validateresources(request)
	accessToken, err := getAccessToken()
	if err != nil {
		return nil, err
	}
	vcall.Header().Add("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	vcall.Header().Add("X-Goog-User-Project", vc.ProjectID)
	vres, err := vcall.Do()
	if err != nil {
		return nil, err
	}
	if vres.HTTPStatusCode != http.StatusOK {
		return nil, fmt.Errorf("expect status code 200, but got %d", vres.HTTPStatusCode)
	}
	return vres, nil
}

// RetrieveOperation retrieves an operation.
func (vc *ValidationClient) RetrieveOperation(name string) (*v1.Operation, error) {
	ocall := vc.Service.Operations.Get(name)
	accessToken, err := getAccessToken()
	if err != nil {
		return nil, err
	}
	ocall.Header().Add("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	ocall.Header().Add("X-Goog-User-Project", vc.ProjectID)
	op, err := ocall.Do()
	if err != nil {
		return nil, err
	}
	if op.HTTPStatusCode != http.StatusOK {
		return nil, fmt.Errorf("expect status code 200, but got %d", op.HTTPStatusCode)
	}
	return op, nil
}

// RetrieveViolationsFromOperation retrieves the violations from an operation.
func (vc *ValidationClient) RetrieveViolationsFromOperation(operation *v1.Operation) ([]*v1.Violation, error) {
	if operation == nil || len(operation.Response) == 0 {
		return nil, nil
	}
	resp := &v1.ValidateResponse{}
	if err := json.Unmarshal(operation.Response, resp); err != nil {
		return nil, err
	}
	return resp.Violations, nil
}
