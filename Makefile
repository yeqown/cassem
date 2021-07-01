GOCMD=CGO_ENABLED=0 GOARCH=amd64 GOOS=darwin go

cassemdb.build:
	${GOCMD} build -o cassemdb ./cmd/cassemdb

cassemdb.run: cassemdb.build
	DEBUG=1 ./cassemdb --conf=./debugdata/cassemdb1.toml > ./debugdata/d1/cassemdb.log &
	sleep 5
	DEBUG=1 ./cassemdb --conf=./debugdata/cassemdb2.toml > ./debugdata/d2/cassemdb.log &
	DEBUG=1 ./cassemdb --conf=./debugdata/cassemdb3.toml > ./debugdata/d3/cassemdb.log &

cassemdb.kill:
	kill -9 $(ps -ef | grep cassemdb | awk '{print $2}')

cassemdb.clear:
	@ rm -fr ./debugdata/d{1,2,3}/{raft.db,cassemdb.kv,cassemdb.log}
	@ rm -fr ./debugdata/d{1,2,3}/snapshots

build-cassemadm:
	${GOCMD} build -o cassemadm ./cmd/cassemadm

build-cassemagent:
	${GOCMD} build -o cassemagent ./cmd/cassemagent

clear:
	@ rm ./cassemdb || rm ./cassemadm || rm ./cassemagent