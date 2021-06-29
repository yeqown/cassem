GOCMD=CGO_ENABLED=0 GOARCH=amd64 GOOS=darwin go

build-cassemdb:
	${GOCMD} build -o cassemdb ./cmd/cassemdb

run-cassemdb: build-cassemdb
	./cassemdb \
		--raft-base="./debugdata/d1" \
		--id="d1" \
		--http-listen="127.0.0.1:2021" \
		--bind="127.0.0.1:3021" \
		--join=""

build-cassemadm:
	${GOCMD} build -o cassemadm ./cmd/cassemadm

build-cassemagent:
	${GOCMD} build -o cassemagent ./cmd/cassemagent

clear:
	@ rm ./cassemdb || rm ./cassemadm || rm ./cassemagent