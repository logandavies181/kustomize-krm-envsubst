package fieldtype

import (
	"errors"
	"fmt"
	"io"
	"net/http"
)

type httpRegistry struct {
	client http.Client
}

func newHttpRegistry() httpRegistry {
	return httpRegistry{
		client: http.Client{},
	}
}

func (hr httpRegistry) DownloadSchema(kind, apiVersion, k8sVersion string) ([]byte, error) {
	nativeRegUrlBase := "https://raw.githubusercontent.com/yannh/kubernetes-json-schema/master"
	// crdsRegUrlBase := "https://raw.githubusercontent.com/datreeio/CRDs-catalog/main/{{.Group}}/{{.ResourceKind}}_{{.ResourceAPIVersion}}.json"

	for _, v := range []string {
		fmt.Sprintf("%s/%s", nativeRegUrlBase, yannhFname(kind, apiVersion, k8sVersion)),
		// fmt.Sprintf("%s/%s", nativeRegUrlBase, yannhFname(kind, apiVersion, k8sVersion)),
	} {
		data, err := hr.downloadSchema(v)
		if err == nil {
			return data, nil
		}
	}

	return nil, newNotFoundError(fmt.Errorf("couldn't find resource from http registries"))
}

func (hr httpRegistry) downloadSchema(url string) ([]byte, error) {
	resp, err := hr.client.Get(url)
	if err != nil {
		msg := fmt.Sprintf("failed downloading schema at %s: %s", url, err)
		return nil, errors.New(msg)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		msg := fmt.Sprintf("could not find schema at %s", url)
		return nil, newNotFoundError(errors.New(msg))
	}

	if resp.StatusCode != http.StatusOK {
		msg := fmt.Sprintf("error while downloading schema at %s - received HTTP status %d", url, resp.StatusCode)
		return nil, fmt.Errorf("%s", msg)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		msg := fmt.Sprintf("failed parsing schema from %s: %s", url, err)
		return nil, errors.New(msg)
	}

	return body, nil
}
