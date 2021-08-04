GOCMD=CGO_ENABLED=0 GOARCH=amd64 GOOS=darwin go

cassemdb.build:
	${GOCMD} build -o cassemdb ./cmd/cassemdb

cassemdb.run: cassemdb.build
	- mkdir ./debugdata/{d1,d2,d3}
	DEBUG=1 ./cassemdb --conf=./examples/cassemdb/cassemdb1.toml > ./debugdata/d1/cassemdb.log &
	sleep 2
	DEBUG=1 ./cassemdb --conf=./examples/cassemdb/cassemdb2.toml > ./debugdata/d2/cassemdb.log &
	DEBUG=1 ./cassemdb --conf=./examples/cassemdb/cassemdb3.toml > ./debugdata/d3/cassemdb.log &

cassemdb.kill:
	kill -9 $(ps -ef | grep cassemdb | awk '{print $2}')

cassemdb.clear:
	- rm -fr ./debugdata/d{1,2,3}/{raft.db,cassemdb.log,cassemdb.kv,snapshots}

cassemadm.build:
	${GOCMD} build -o cassemadm ./cmd/cassemadm

cassemadm.run: cassemadm.build
	DEBUG=1 ./cassemadm --conf=./examples/cassemadm/cassemadm.toml

cassemagent.build:
	${GOCMD} build -o cassemagent ./cmd/cassemagent

clear:
	- rm ./cassemdb || rm ./cassemadm || rm ./cassemagent