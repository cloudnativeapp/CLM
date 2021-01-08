FROM golang:1.14.4-alpine3.11 as builder
ARG VERSION
ARG GIT_REVISION
ARG BUILD_DATE

RUN apk add --no-cache git
WORKDIR /go/src/
RUN go get github.com/go-delve/delve/cmd/dlv

WORKDIR /workspace
ENV GOPATH /workspace
RUN mkdir -p /workspace/src/cloudnativeapp/clm
# Copy the Go Modules manifests
COPY go.mod /workspace/src/cloudnativeapp/clm/go.mod
COPY go.sum /workspace/src/cloudnativeapp/clm/go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN cd /workspace/src/cloudnativeapp/clm/ && go mod download

COPY main.go /workspace/src/cloudnativeapp/clm/main.go
COPY api /workspace/src/cloudnativeapp/clm/api/
COPY controllers /workspace/src/cloudnativeapp/clm/controllers/
COPY internal /workspace/src/cloudnativeapp/clm/internal/
COPY pkg /workspace/src/cloudnativeapp/clm/pkg/

ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on GIT_REVISION=$GIT_REVISION BUILD_DATE=$BUILD_DATE

# Build
RUN cd /workspace/src/cloudnativeapp/clm/ && go build -gcflags="all=-N -l" -a -o manager main.go

FROM alpine:3.10.2
WORKDIR /
COPY --from=builder /workspace/src/cloudnativeapp/clm/manager .
COPY --from=builder /go/bin/dlv /
EXPOSE 40000
RUN apk add tzdata
RUN echo "hosts: files dns" > /etc/nsswitch.conf
RUN cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime
RUN echo 'Asia/Shanghai' >/etc/timezone
RUN apk add curl
RUN apk del tzdata
ENTRYPOINT ["/manager"]