FROM golang:1.15-alpine3.12 as builder

# Copy the go source
COPY main.go /go/src/destination/main.go
WORKDIR /go/src/destination
RUN go env -w GOPROXY=https://goproxy.cn,direct && \
    go mod init && \
    go mod download

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o destination main.go

FROM alpine:3.12
WORKDIR /
COPY --from=builder /go/src/destination .
# USER 65532:65532
EXPOSE 8081

# ENTRYPOINT ["/destination"]
CMD ["/destination"]