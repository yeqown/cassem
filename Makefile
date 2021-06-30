GOCMD=CGO_ENABLED=0 GOARCH=amd64 GOOS=darwin go

build-cassemdb:
	${GOCMD} build -o cassemdb ./cmd/cassemdb

run-cassemdb: build-cassemdb
	./cassemdb --conf=./debugdata/cassemdb1.toml

build-cassemadm:
	${GOCMD} build -o cassemadm ./cmd/cassemadm

build-cassemagent:
	${GOCMD} build -o cassemagent ./cmd/cassemagent

clear:
	@ rm ./cassemdb || rm ./cassemadm || rm ./cassemagent