package allowlist

import (
	"embed"

	"github.com/blackzlq/k8s-validation-service-integration-demo/validate"
	v1 "gke-internal.googlesource.com/k8ssecurityvalidation_pa/client.git/v1"
)

var (
	FocusComponentMap = map[string]string{
		"hub":                              "^hub$",
		"hub-db-dir":                       "^hub-db-dir$",
		"jupyter-sa":                       "^jupyter-sa.*",
		"kuberay-operator":                 "^kuberay-operator$",
		"kuberay-operator-leader-election": "^kuberay-operator-leader-election$",
		"proxy-public":                     "^proxy-public$",
		"rag-frontend":                     "^rag-frontend$",
		"ray-cluster-kuberay":              "^ray-cluster-kuberay$",
		"ray-cluster-kuberay-head":         "^ray-cluster-kuberay-head-[a-zA-Z0-9]+$",
		"ray-cluster-kuberay-head-svc":     "^ray-cluster-kuberay-head-svc$",
		"ray-dashboard-iap-config":         "^ray-dashboard-iap-config$",
		"ray-dashboard-ingress":            "^ray-dashboard-ingress$",
		"ray-dashboard-managed-cert":       "^ray-dashboard-managed-cert$",
		"ray-dashboard-secret":             "^ray-dashboard-secret$",
		"ray-monitoring":                   "^ray-monitoring$",
		"ray-operator-leader":              "^ray-operator-leader$",
		"ray-sa":                           "ray-sa-.*",
	}
	//go:embed category/**/*
	allowList embed.FS

	allowedResourceKeysMap = validate.AllowedResourceKeysMap(allowList)
)

func FindFocusComponent(name string) (string, bool) {
	return validate.FindFocusComponent(name, FocusComponentMap)
}

func PurifyViolation(v *v1.Violation) *v1.Violation {
	return v
}

func ShouldReport(v *v1.Violation) bool {
	return validate.ShouldReport(v, allowedResourceKeysMap)
}
