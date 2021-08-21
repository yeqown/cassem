#!/bin/bash

# go tool pprof -http=:8888 -seconds=30 http://localhost:2021/debug/pprof/profile

# benchmark
ghz \
  --insecure \
  --async \
  --proto ~/projects/opensource/cassem/internal/cassemdb/api/cassemdb.api.proto \
  -i ~/projects/opensource/cassem/thirdparty \
  --call cassem.db.KV/GetKV \
  -c 10 -n 1000 --rps 100 \
  -d '{
  "key":"root/elements/app/env/ele1/v1"
  }' \
  127.0.0.1:2021

#Summary:
#  Count:	10000
#  Total:	5.00 s
#  Slowest:	98.66 ms
#  Fastest:	0.37 ms
#  Average:	4.21 ms
#  Requests/sec:	1999.35
#
#Response time histogram:
#  0.373  [1]	|
#  10.202 [9282]	|∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎∎
#  20.031 [201]	|∎
#  29.860 [83]	|
#  39.689 [81]	|
#  49.519 [96]	|
#  59.348 [76]	|
#  69.177 [67]	|
#  79.006 [43]	|
#  88.836 [37]	|
#  98.665 [33]	|
#
#Latency distribution:
#  10 % in 0.48 ms
#  25 % in 0.54 ms
#  50 % in 0.80 ms
#  75 % in 1.73 ms
#  90 % in 5.53 ms
#  95 % in 21.61 ms
#  99 % in 73.42 ms
#
#Status code distribution:
#  [OK]   10000 responses
