package main

import (
	"fmt"
	"os"

	"github.com/drone/envsubst"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/fn/framework/command"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func walkSequenceNode(in *yaml.RNode) error {
	_, err := filter(in)
	if err != nil {
		return err
	}

	return nil
}

func walkMapNode(in *yaml.MapNode) error {
	_, err := filter(in.Value)
	if err != nil {
		return err
	}

	return nil
}

func filter(in *yaml.RNode) (*yaml.RNode, error) {
	if in.IsNil() {
		return nil, nil
	}

	switch y := in.YNode().Kind; y {
	case yaml.MappingNode:
		err := in.VisitFields(walkMapNode)
		if err != nil {
			return nil, fmt.Errorf("Could not visit fields: %v", err)
		}
		return in, nil
	case yaml.SequenceNode:
		err := in.VisitElements(walkSequenceNode)
		if err != nil {
			return nil, fmt.Errorf("Could not visit elements: %v", err)
		}

		return in, nil
	case yaml.ScalarNode:
		str, err := in.String()
		if err != nil {
			return nil, fmt.Errorf("Could not parse node into string: %v", err)
		}

		substed, err := envsubst.EvalEnv(str)
		if err != nil {
			return nil, fmt.Errorf("Could not envsubt: %v", err)
		}

		if substed == str {
			return in, nil
		}

		strNode, err := yaml.Parse(substed)
		if err != nil {
			return nil, fmt.Errorf("Could not parse node after envsubsting: %v", err)
		}

		out, err := in.Pipe(yaml.Set(strNode))
		if err != nil {
			return nil, err
		}

		return out, nil

	case yaml.AliasNode, yaml.DocumentNode:
		fallthrough
	default:
		panic(fmt.Sprintf("Unknown Kind: %v", y))
	}
}

func main() {
	fn := func(items []*yaml.RNode) ([]*yaml.RNode, error) {
		for i := range items {
			err := items[i].PipeE(yaml.FilterFunc(filter))
			if err != nil {
				return nil, fmt.Errorf("kustomize-krm-envsubt: %v", err)
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
