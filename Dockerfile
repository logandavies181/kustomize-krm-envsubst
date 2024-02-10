FROM golang:1.22 as builder
ENV CGO_ENABLED=0
WORKDIR /build
COPY . .
RUN go build

FROM alpine:latest
COPY --from=builder /build/kustomize-krm-envsubst /
ENTRYPOINT ["/kustomize-krm-envsubst"]
