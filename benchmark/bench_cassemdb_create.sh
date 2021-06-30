#!/bin/bash

# go tool pprof -http=:8888 -seconds=25 http://localhost:2021/debug/pprof/profile

# create kv
for i in {1..10000}
do
  val="{\"key\": \"bench/$i\",\"value\": \"`echo "my value is: $i" | base64`\"}"
  echo $val
  curl -X POST -H "Content-Type: application/json" http://localhost:2021/api/kv -d "$val"
done