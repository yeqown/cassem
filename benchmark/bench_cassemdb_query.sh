#!/bin/bash

# go tool pprof -http=:8888 -seconds=30 http://localhost:2021/debug/pprof/profile

# create kv
for i in {1..10000}
do
  curl -X GET "http://localhost:2021/api/kv?key=bench/$i"
done