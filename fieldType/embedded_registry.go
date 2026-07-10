package fieldtype

import (
	"embed"
	"fmt"
	"io"
	"io/fs"
)

var schemafs embed.FS

type embeddedRegistry struct {
	fs  fs.FS
}

func newEmbeddedRegistry(fs fs.FS) embeddedRegistry {
	return embeddedRegistry{
		fs: fs,
	}
}

func (er embeddedRegistry) DownloadSchema(kind, apiVersion, k8sVersion string) ([]byte, error) {
	if kind == "" {
		return nil, newNotFoundError(fmt.Errorf("No kind supplied"))
	}

	fname := yannhFname(kind, apiVersion, k8sVersion)

	f, err := er.fs.Open(fname)
	if err != nil {
		return nil, newNotFoundError(fmt.Errorf("not found in embedded registry"))
	}
	defer f.Close()

	return io.ReadAll(f)
}
