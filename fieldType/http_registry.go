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

func (hr httpRegistry) DownloadSchema(kind, apiVersion string) ([]byte, error) {
	nativeRegUrlBase := "https://raw.githubusercontent.com/yannh/kubernetes-json-schema/master"
	crdsRegUrlBase := "https://raw.githubusercontent.com/datreeio/CRDs-catalog/main"

	yannhTarget := fmt.Sprintf("%s/%s", nativeRegUrlBase, yannhFname(kind, apiVersion))
	datreeTarget := fmt.Sprintf("%s/%s", crdsRegUrlBase, datreeFname(kind, apiVersion))

	var targets []string
	if crdsFirst {
		targets = []string{datreeTarget, yannhTarget}
	} else {
		targets = []string{yannhTarget, datreeTarget}
	}

	for _, v := range targets {
		data, err := hr.downloadSchema(v)
		switch err.(type) {
		case notFoundError:
			continue
		case nil:
			return data, nil
		default:
			return nil, err
		}
	}

	return nil, newNotFoundError(fmt.Errorf("couldn't find resource from http registries"))
}

// The following is modified from yannh/kubeconform.
//   - We return the raw []byte rather than unmarshalling.
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
