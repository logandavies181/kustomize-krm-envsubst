FROM scratch
COPY kustomize-krm-envsubst /
ENTRYPOINT ["/kustomize-krm-envsubst"]
