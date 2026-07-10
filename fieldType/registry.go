package fieldtype

import (
	"fmt"
	"strings"
)

type registry interface {
	DownloadSchema(kind, apiVersion, k8sVersion string) ([]byte, error)
}

type notFoundError struct {
	error
}

func newNotFoundError(err error) notFoundError {
	return notFoundError{err}
}

func yannhFname(kind, apiVersion, k8sVersion string) string {
	parts := []string{
		kind,
	}
	if apiVersion != "" {
		parts = append(parts, strings.ReplaceAll(apiVersion, "/", "-"))
	}
	return strings.ToLower(
		fmt.Sprintf(
			"master-standalone-strict/%s.json",
			strings.Join(parts, "-"),
		),
	)
}
