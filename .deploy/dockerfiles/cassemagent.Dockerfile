# # build stage
# FROM golang:1.17 as stage.build
#
# WORKDIR /tmp/build
# COPY . .
# # RUN git rev-parse  --show-toplevel
# ENV GOPROXY https://goproxy.cn,direct
# RUN go mod download && \
#     go build -o cassemagent \
#             -ldflags "-s \
#                       -X main.Version=`git tag --list | tail -n 1` \
#                       -X main.BuildTime=`TZ=UTC date -u '+%Y-%m-%dT%H:%M:%SZ'` \
#                       -X main.GitHash=`git rev-parse HEAD`" \
#             ./cmd/cassemagent
#
# # package image stage
# FROM alpine as stage.pack

FROM alpine

WORKDIR /app/cassemagent
ENV APP_PATH /app/cassemagent
COPY ./cassemagent $APP_PATH

CMD ["./cassemagent", "-conf", "./configs/cassemagent.toml"]