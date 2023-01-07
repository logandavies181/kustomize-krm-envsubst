package main

import (
	"fmt"
	"os"

	"github.com/logandavies181/envsubst"

	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/fn/framework/command"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

type Config struct {
	envMapping envsubst.AdvancedMapping

	AllowEmpty   bool     `yaml:"allowEmpty" json:"allowEmpty"`
	ExcludedVars []string `yaml:"excludedVars" json:"excludedVars"`
	IncludedVars []string `yaml:"includedVars" json:"includedVars"`
}

func isEmpty(str string) bool {
	switch str {
	case "", `""`, "''", "\n":
		return true
	default:
		return false
	}
}

func contains(list []string, str string) bool {
	for _, v := range list {
		if str == v {
			return true
		}
	}

	return false
}

func (c Config) walkSequenceNode(in *yaml.RNode) error {
	_, err := c.Filter(in)
	if err != nil {
		return err
	}

	return nil
}

func (c Config) walkMapNode(in *yaml.MapNode) error {
	_, err := c.Filter(in.Value)
	if err != nil {
		return err
	}

	return nil
}

func (c Config) Filter(in *yaml.RNode) (*yaml.RNode, error) {
	if in.IsNil() {
		return nil, nil
	}

	switch y := in.YNode().Kind; y {
	case yaml.MappingNode:
		err := in.VisitFields(c.walkMapNode)
		if err != nil {
			return nil, fmt.Errorf("Could not visit fields: %v", err)
		}
		return in, nil
	case yaml.SequenceNode:
		err := in.VisitElements(c.walkSequenceNode)
		if err != nil {
			return nil, fmt.Errorf("Could not visit elements: %v", err)
		}

		return in, nil
	case yaml.ScalarNode:
		str, err := in.String()
		if err != nil {
			return nil, fmt.Errorf("Could not parse node into string: %v", err)
		}

		substed, err := envsubst.EvalAdvanced(str, envsubst.AdvancedMapping(c.envMapping))
		if err != nil {
			return nil, fmt.Errorf("Could not envsubst: %v", err)
		}

		if substed == str {
			return in, nil
		}

		if isEmpty(substed) {
			if !c.AllowEmpty {
				return nil, fmt.Errorf(
					"Value %s evaluated to empty string. Did you forget to set an environment variable?",
					str)
			}

			substed = `""`
		}

		strNode, err := yaml.Parse(substed)
		if err != nil {
			fmt.Fprintln(os.Stderr, substed)
			fmt.Fprintln(os.Stderr, len(substed))
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
	config := new(Config)
	fn := func(items []*yaml.RNode) ([]*yaml.RNode, error) {
		config.envMapping = func(str string, nodeInfo envsubst.NodeInfo) (string, bool) {
			if len(config.IncludedVars) == 0 {
				if contains(config.ExcludedVars, str) {
					return nodeInfo.Orig(), false
				}

				return str, true
			}

			if !contains(config.IncludedVars, str) {
				return nodeInfo.Orig(), false
			}

			return str, true
		}

		for i := range items {
			err := items[i].PipeE(config)
			if err != nil {
				return nil, fmt.Errorf("kustomize-krm-envsubst: %v ", // leave a space because kustomize doesn't
					err)
			}
		}
		return items, nil
	}
	p := framework.SimpleProcessor{Config: config, Filter: kio.FilterFunc(fn)}
	cmd := command.Build(p, command.StandaloneDisabled, false)
	command.AddGenerateDockerfile(cmd)
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
