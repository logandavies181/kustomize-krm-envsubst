# kustomize-krm-envsubst

This is an [Exec KRM function](https://kubectl.docs.kubernetes.io/guides/extending_kustomize/exec_krm_functions/) for
Kustomize. It applies envsubst as per this package: https://github.com/drone/envsubst on the configuration files

Needs to be run with `--enable-alpha-plugins --enable-exec`
