## cassemdb 

cassemd storage component which is distributed like ETCD.

```shell
GET rootKey/leafKey

SET rootKey/leafKey value

WATCH key
#1 key'value changed: revision=xxxx value=xxx
```

### Get Started

```shell
cassemdb \
	--listen=8080 \
	--raft-listen=8081 \
	--id=1 \ 
	--dir=./cassemdb-data1 \ 
	--join=""
	
	
curl -X GET http://localhost:8080/keys/r?key=a/b/c

curl -X POST https://localhost:8080/keys/w?key=a/b/c&value=10
```