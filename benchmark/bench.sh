#!/bin/bash

# get container detail
go-wrk -n=1000 -c=100 -t=10 http://localhost:2021/api/namespaces/ns/container-1

# paging namespace
go-wrk -n=1000 -c=100 -t=10 http://localhost:2021/api/namespaces?limit=1&offset=0&key=ns

# paging pairs
go-wrk -n=1000 -c=100 -t=10 http://localhost:2021/api/namespaces/ns/pairs?limit=20&offset=0