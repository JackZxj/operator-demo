FROM golang:1.15-alpine3.12 as builder

WORKDIR /workspace

# Copy the go source
COPY main.go main.go

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o source main.go

FROM alpine:3.12
WORKDIR /
COPY --from=builder /workspace/source .
# USER 65532:65532
EXPOSE 8080

ENTRYPOINT ["/source"]