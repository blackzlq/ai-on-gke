package validate

import (
	"context"

	v1 "gke-internal.googlesource.com/k8ssecurityvalidation_pa/client.git/v1"
)

type RequestObject struct {
	Content      *v1.Content
	ResourceName string
}

type ResultObject struct {
	Violations   []*v1.Violation
	ResourceName string
}

// Fetcher interface for fetching YAML files
type Fetcher interface {
	PrepareValidateRequestContent(ctx context.Context) ([]RequestObject, error)
}

// ResultHandler interface for handling results
type ResultHandler interface {
	HandleResult(ctx context.Context, objects []ResultObject) error
}
