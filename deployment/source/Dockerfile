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

# 模式1，可以通过docker run -e更新环境变量，容器实际运行 /source
# ENV SOURCE=/source
# ENTRYPOINT ["/source"]

# 模式2，无需在dockerfile中指定env，可以直接读取run -e的环境变量，容器实际运行 /bin/sh -c /source
# ENTRYPOINT /source

# 模式3，无需在dockerfile中指定env，可以直接读取run -e的环境变量，容器实际运行 /source
CMD ["/source"]
