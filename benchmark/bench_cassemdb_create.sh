#!/bin/bash

# analyze go program performance profile.
# go tool pprof -http=:8888 -seconds=25 http://localhost:2021/debug/pprof/profile

#if [ ! -x 'ghz' ]; then
#  echo "ghz not installed"
#  echo "install command: brew install ghz"
#  exit 0
#fi

# benchmark
ghz \
  --insecure \
  --async \
  --proto ~/projects/opensource/cassem/internal/cassemdb/api/cassemdb.api.proto \
  -i ~/projects/opensource/cassem/thirdparty \
  --call cassem.db.KV/SetKV \
  -c 5 -n 500 --rps 50 \
  -d '{
  "key":"tmp/benchmark/write_test",
  "isDir": false,
  "val":"MTIzMTIzMTIzMTIzMTIzMTIzMTIzMTIzMTIzMTIzMTIzMTIzMTIzMTIzMTIzMTIzMTIzMTIzMTIzMTIzMTIzMTIzMTIzMTIzMTIzMTIzMTIzMTIzMTIzMTIzMTIzMTIzMTIzMTIzMTIzMTIzMTIzMTIzMTIzMTIzMTIzMTIzMTIzMTIzMTIzMTIzMTIzMTIzCg==",
  "ttl":30,
  "overwrite": true
  }' \
  127.0.0.1:2021

# QPS ~ 20