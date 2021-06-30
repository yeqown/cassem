#!/bin/sh

curl -X POST -H "Content-Type:application/json" "http://127.0.0.1:2021/api/kv" -d '
{
  "key": "appid/cluster/env/itermkey-rev1",
  "value": "SSdtIHN0cmluZw=="
}'