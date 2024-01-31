# kustomize-krm-envsubst

This is an [Exec KRM function](https://kubectl.docs.kubernetes.io/guides/extending_kustomize/exec_krm_functions/) for
Kustomize. It applies envsubst as per this package: https://github.com/drone/envsubst on the configuration files

## Installation

Build from source:

```bash
go install github.com/logandavies181/kustomize-krm-envsubst@latest
```

Or check out [releases](https://github.com/logandavies181/kustomize-krm-envsubst/releases)

## Usage

### As an Exec KRM function

Install this binary as above.

Add a transformer to your kustomize configuration

```yaml
# kustomization.yaml
resources:
- secret.yaml
transformers:
- transformer.yaml
```

```yaml
# transformer.yaml
apiVersion: kustomize-krm-envsubst/v1alpha
kind: Envsubst
metadata:
  name: envsubst
  annotations:
    config.kubernetes.io/function: |
      exec:
        # ~ is not expanded by kustomize :(
        path: /path/to/kustomize-krm-envsubst
#spec:
#  excludedVariableNames: [] # used to denylist certain env var names from being injected    
#  includedVariableNames: [] # used to enumerate the list of env var names to inject
```

Inject environment variables into your manifests!

```yaml
# secret.yaml
apiVersion: v1
kind: Secret
metadata:
  name: super-secret-value
# always use stringData for using this plugin with secrets
stringData:
  password: ${PASSWORD}
```

Render your configuration

```bash
# Can only be run with `kustomize` and not `kubectl kustomize`
# see https://github.com/kubernetes-sigs/kustomize/issues/4556#issuecomment-1092095023

kustomize build --enable-alpha-plugins --enable-exec .
```

### As a legacy exec plugin

Use this method to use this plugin with `kubectl kustomize`

Install the binary to
`${XDG_CONFIG_HOME:-~/.config}/kustomize/plugin/kustomize-krm-envsubst/v1alpha/kustomize-krm-envsubst/kustomize-krm-envsubst`

Set up your kustomization.yaml and other files as above but use this config for transformer.yaml:

```yaml
# transformer.yaml
apiVersion: kustomize-krm-envsubst/v1alpha
kind: Envsubst
metadata:
  name: envsubst
#excludedVariableNames: [] # used to denylist certain env var names from being injected    
#includedVariableNames: [] # used to enumerate the list of env var names to inject
```
