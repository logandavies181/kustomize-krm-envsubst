package main

import (
	"os"

	"github.com/drone/envsubst"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/fn/framework/command"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func filter(in *yaml.RNode) (*yaml.RNode, error) {
	str, err := in.String()
	if err != nil {
		return nil, err
	}

	substed, err := envsubst.EvalEnv(str)
	if err != nil {
		return nil, err
	}

	obj, err := yaml.Parse(substed)
	if err != nil {
		return nil, err
	}

	*in = *obj

	return in, nil
}

func main() {
	fn := func(items []*yaml.RNode) ([]*yaml.RNode, error) {
		for i := range items {
			err := items[i].PipeE(yaml.FilterFunc(filter))
			if err != nil {
				return nil, err
			}
		}
		return items, nil
	}
	p := framework.SimpleProcessor{Config: nil, Filter: kio.FilterFunc(fn)}
	cmd := command.Build(p, command.StandaloneDisabled, false)
	command.AddGenerateDockerfile(cmd)
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
