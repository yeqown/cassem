#!/bin/bash

# Preparation:
# 1. make sure open the server debug mode, so that pprof tool is enabled.
# 2. start the RESTful API server `cassemctl -c CONFIG_FILE serve`
# 3. start go pprof web server `go tool pprof -http=localhost:1414 http://localhost:2021/debug/pprof/profile\?seconds\=30`
# 4. execute following benchmark script

# get container detail
go-wrk -n=1000 -c=100 -t=10 http://localhost:2021/api/namespaces/ns/containers/container-1

# paging namespace
go-wrk -n=1000 -c=100 -t=10 http://localhost:2021/api/namespaces?limit=1&offset=0&key=ns

# paging pairs
go-wrk -n=1000 -c=100 -t=10 http://localhost:2021/api/namespaces/ns/pairs?limit=20&offset=0