#!/bin/bash

# go tool pprof -http=:8888 -seconds=25 http://localhost:2021/debug/pprof/profile

# create kv
for i in {1..100}
do
  val="{\"raw\": \"my value is: $i\", \"content_type\": \"application/plaintext\"}"
  #echo $val
  uri="http://localhost:20218/api/apps/app2/envs/env/elements/bench-${i}"
  #echo uri
  curl -X POST -H "Content-Type: application/json" "$uri" -d "$val"
done