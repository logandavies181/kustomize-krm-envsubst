package fieldtype

import (
	"fmt"
	"strings"
)

type registry interface {
	DownloadSchema(kind, apiVersion string) ([]byte, error)
}

type notFoundError struct {
	error
}

func newNotFoundError(err error) notFoundError {
	return notFoundError{err}
}

func yannhFname(kind, apiVersion string) string {
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

func datreeFname(kind, apiVersion string) string {
	parts := strings.Split(apiVersion, "/")
	if len(parts) != 2 {
		return ""
	}

	return strings.ToLower(
		fmt.Sprintf("%s/%s_%s.json", parts[0], kind, parts[1]),
	)
}
