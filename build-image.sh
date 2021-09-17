#!/bin/bash

# compile all app binaries at local machine those will be send into
# docker images which can help the process building images by using
# local machine's go modules cache.

echo "building cassemdb..."
go mod download && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o cassemdb \
            -ldflags "-s \
                      -X main.Version=`git tag --list | tail -n 1` \
                      -X main.BuildTime=`TZ=UTC date -u '+%Y-%m-%dT%H:%M:%SZ'` \
                      -X main.GitHash=`git rev-parse HEAD`" \
            ./cmd/cassemdb

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
docker build -t yeqown/cassemdb:${IMAGE_TAG} -f ./.deploy/dockerfiles/cassemdb.Dockerfile .
docker build -t yeqown/cassemadm:${IMAGE_TAG} -f ./.deploy/dockerfiles/cassemadm.Dockerfile .
docker build -t yeqown/cassemagent:${IMAGE_TAG} -f ./.deploy/dockerfiles/cassemagent.Dockerfile .