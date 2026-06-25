#!/bin/bash

# compile all app binaries at local machine those will be send into
# docker images which can help the process building images by using
# local machine's go modules cache.

echo "building cassemkv..."
go mod download && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o cassemkv \
            -ldflags "-s \
                      -X main.Version=`git tag --list | tail -n 1` \
                      -X main.BuildTime=`TZ=UTC date -u '+%Y-%m-%dT%H:%M:%SZ'` \
                      -X main.GitHash=`git rev-parse HEAD`" \
            ./cmd/cassemkv

echo "building cassemadm..."
go mod download && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64  go build -o cassemadm \
            -ldflags "-s \
                      -X main.Version=`git tag --list | tail -n 1` \
                      -X main.BuildTime=`TZ=UTC date -u '+%Y-%m-%dT%H:%M:%SZ'` \
                      -X main.GitHash=`git rev-parse HEAD`" \
            ./cmd/cassemadm

echo "building cassemagent..."
go mod download && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o cassemagent \
            -ldflags "-s \
                      -X main.Version=`git tag --list | tail -n 1` \
                      -X main.BuildTime=`TZ=UTC date -u '+%Y-%m-%dT%H:%M:%SZ'` \
                      -X main.GitHash=`git rev-parse HEAD`" \
            ./cmd/cassemagent

# build image-all
IMAGE_TAG=$(git tag --list | tail -n 1)
docker build -t yeqown/cassemkv:${IMAGE_TAG} -f ./.deploy/dockerfiles/cassemkv.Dockerfile .
docker build -t yeqown/cassemadm:${IMAGE_TAG} -f ./.deploy/dockerfiles/cassemadm.Dockerfile .
docker build -t yeqown/cassemagent:${IMAGE_TAG} -f ./.deploy/dockerfiles/cassemagent.Dockerfile .