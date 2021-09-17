# # build stage
# FROM golang:1.17 as stage.build
#
# WORKDIR /tmp/build
# COPY . .
# # RUN git rev-parse  --show-toplevel
# ENV GOPROXY https://goproxy.cn,direct
# RUN go mod download && \
#     go build -o cassemdb \
#             -ldflags "-s \
#                       -X main.Version=`git tag --list | tail -n 1` \
#                       -X main.BuildTime=`TZ=UTC date -u '+%Y-%m-%dT%H:%M:%SZ'` \
#                       -X main.GitHash=`git rev-parse HEAD`" \
#             ./cmd/cassemdb
#
# # package image stage
# FROM alpine as stage.pack

FROM alpine

WORKDIR /app/cassemdb
ENV APP_PATH /app/cassemdb
COPY ./cassemdb $APP_PATH

CMD ["./cassemdb", "-conf", "./configs/cassemdb.toml"]