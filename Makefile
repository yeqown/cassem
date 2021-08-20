GOCMD=CGO_ENABLED=0 GOARCH=amd64 GOOS=darwin go1.17

cassemdb.build:
	${GOCMD} build 	-o cassemdb \
					-ldflags "-s \
							  -X main.Version=`git tag --list | tail -n 1` \
							  -X main.BuildTime=`TZ=UTC date -u '+%Y-%m-%dT%H:%M:%SZ'` \
							  -X main.GitHash=`git rev-parse HEAD`" \
					./cmd/cassemdb

cassemdb.run: cassemdb.build cassemdb.kill
	- mkdir ./debugdata/{d1,d2,d3}
	DEBUG=1 ./cassemdb --conf=./examples/cassemdb/cassemdb1.toml > ./debugdata/d1/cassemdb.log 2>&1 & \
		echo $$! >> cassemdb.pids
	sleep 2
	DEBUG=1 ./cassemdb --conf=./examples/cassemdb/cassemdb2.toml > ./debugdata/d2/cassemdb.log 2>&1 & \
		echo $$! >> cassemdb.pids
	DEBUG=1 ./cassemdb --conf=./examples/cassemdb/cassemdb3.toml > ./debugdata/d3/cassemdb.log 2>&1 & \
		echo $$! >> cassemdb.pids

cassemdb.kill:
	@ echo "clearing running cassemdb process from cassemdb.pids"
	@ if [ -f "cassemdb.pids" ]; then \
		cat cassemdb.pids | xargs kill -9;\
	fi
	- rm cassemdb.pids
	#
	# If cassemdb process is not killed as expected, you can try following command:
	#
	# 1: kill -9 $$(ps -ef | grep cassemdb | awk '{print $2}')
	# 2: jobs -l | grep cassemdb | awk '{print $3}' | xargs kill -9

cassemdb.clear:
	- rm -fr ./cassemdb.pids
	- rm -fr ./debugdata/d{1,2,3}/{raft.db,cassemdb.log,cassemdb.kv,snapshots}

cassemadm.build:
	${GOCMD} build 	-o cassemadm \
					-ldflags "-s \
							  -X main.Version=`git tag --list | tail -n 1` \
							  -X main.BuildTime=`TZ=UTC date -u '+%Y-%m-%dT%H:%M:%SZ'` \
							  -X main.GitHash=`git rev-parse HEAD`" \
					./cmd/cassemadm

cassemadm.run: cassemadm.build
	DEBUG=1 ./cassemadm --conf=./examples/cassemadm/cassemadm.toml

cassemagent.build:
	${GOCMD} build 	-o cassemagent \
					-ldflags "-s \
							  -X main.Version=`git tag --list | tail -n 1` \
							  -X main.BuildTime=`TZ=UTC date -u '+%Y-%m-%dT%H:%M:%SZ'` \
							  -X main.GitHash=`git rev-parse HEAD`" \
					./cmd/cassemagent

cassemagent.run: cassemagent.build
	DEBUG=1 ./cassemagent --conf=./examples/cassemagent/cassemagent.toml

build-all: cassemadm.build cassemagent.build cassemdb.build

proto-all:
	make -C ./internal/cassemdb/api
	make -C ./internal/concept
	make -C ./internal/cassemagent/api

clear:
	- rm ./cassemdb || rm ./cassemadm || rm ./cassemagent