# # build stage
# FROM golang:1.17 as stage.build
#
# WORKDIR /tmp/build
# COPY . .
# # RUN git rev-parse  --show-toplevel
# ENV GOPROXY https://goproxy.cn,direct
# RUN go mod download && \
#     go build -o cassemadm \
#             -ldflags "-s \
#                       -X main.Version=`git tag --list | tail -n 1` \
#                       -X main.BuildTime=`TZ=UTC date -u '+%Y-%m-%dT%H:%M:%SZ'` \
#                       -X main.GitHash=`git rev-parse HEAD`" \
#             ./cmd/cassemadm
#
# # package image stage
# FROM alpine as stage.pack

FROM alpine

WORKDIR /app/cassemadm
ENV APP_PATH /app/cassemadm
COPY ./cassemadm $APP_PATH

CMD ["./cassemadm", "-conf", "./configs/cassemadm.toml"]