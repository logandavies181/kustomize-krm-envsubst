package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/logandavies181/envsubst"
	"github.com/logandavies181/go-buildversion"

	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/fn/framework/command"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

var version string // goreleaser will set this

type Config struct {
	envMapping envsubst.AdvancedMapping

	AllowEmpty   bool     `yaml:"allowEmpty" json:"allowEmpty"`
	ExcludedVars []string `yaml:"excludedVariableNames" json:"excludedVariableNames"`
	IncludedVars []string `yaml:"includedVariableNames" json:"includedVariableNames"`
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
			return nil, err
		}
		return in, nil
	case yaml.SequenceNode:
		err := in.VisitElements(c.walkSequenceNode)
		if err != nil {
			return nil, err
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
					"Value `%s` evaluated to empty string. Did you forget to set an environment variable?",
					strings.TrimSuffix(str, "\n"))
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

	// This is a hack, but legacy exec plugins get config from the first arg.
	// The framework is going to do its own thing here, but let's try read the first arg
	// and parse it if it exists and is a file
	if len(os.Args) > 1 {
		fname := os.Args[1]
		fdata, err := os.ReadFile(fname)
		if err == nil {
			var c Config
			err = yaml.Unmarshal(fdata, &c)
			if err == nil {
				config.AllowEmpty = c.AllowEmpty
				config.IncludedVars = c.IncludedVars
				config.ExcludedVars = c.ExcludedVars
			}
		}
	}

	for _, arg := range os.Args {
		fmt.Fprintln(os.Stderr, arg)
	}

	fn := func(items []*yaml.RNode) ([]*yaml.RNode, error) {
		config.envMapping = func(str string, nodeInfo envsubst.NodeInfo) (string, bool) {
			// IncludedVars and ExcludedVars are mutually exclusive
			// IncludedVars takes precedent
			// TODO: readme

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
	version, _ := buildversion.BuildVersionShortE(version)
	cmd.Version = version

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
