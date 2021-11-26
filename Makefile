GOCMD=CGO_ENABLED=0 GOARCH=amd64 GOOS=darwin go1.17
GOCMD_LINUX=CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go1.17

cassemdb.build:
	${GOCMD} build 	-o cassemdb \
					-ldflags "-s \
							  -X main.Version=`git tag --list | tail -n 1` \
							  -X main.BuildTime=`TZ=UTC date -u '+%Y-%m-%dT%H:%M:%SZ'` \
							  -X main.GitHash=`git rev-parse HEAD`" \
					./cmd/cassemdb

cassemdb.build-linux:
	${GOCMD_LINUX} build 	-o cassemdb \
					-ldflags "-s \
							  -X main.Version=`git tag --list | tail -n 1` \
							  -X main.BuildTime=`TZ=UTC date -u '+%Y-%m-%dT%H:%M:%SZ'` \
							  -X main.GitHash=`git rev-parse HEAD`" \
					./cmd/cassemdb

cassemdb.run: cassemdb.build cassemdb.kill
	- mkdir ./debugdata/{1,2,3}
	DEBUG=1 ./cassemdb \
		--conf=./examples/cassemdb/cassemdb.toml \
		--endpoint=127.0.0.1:2021 \
		--raft.cluster=http://127.0.0.1:3021,http://127.0.0.1:3022,http://127.0.0.1:3023 \
		--raft.bind=http://127.0.0.1:3021 \
		--storage="./debugdata/1" > ./debugdata/1/cassemdb.log 2>&1 & \
		echo $$! >> cassemdb.pids
	sleep 2
	DEBUG=1 ./cassemdb \
		--conf=./examples/cassemdb/cassemdb.toml \
		--endpoint=127.0.0.1:2022 \
		--raft.cluster=http://127.0.0.1:3021,http://127.0.0.1:3022,http://127.0.0.1:3023 \
		--raft.bind=http://127.0.0.1:3022 \
		--storage="./debugdata/2" > ./debugdata/2/cassemdb.log 2>&1 & \
		echo $$! >> cassemdb.pids
	DEBUG=1 ./cassemdb \
		--conf=./examples/cassemdb/cassemdb.toml \
		--endpoint=127.0.0.1:2023 \
		--raft.cluster=http://127.0.0.1:3021,http://127.0.0.1:3022,http://127.0.0.1:3023 \
		--raft.bind=http://127.0.0.1:3023 \
		--storage="./debugdata/3" > ./debugdata/3/cassemdb.log 2>&1 & \
		echo $$! >> cassemdb.pids

cassemdb.kill:
	@ echo "clearing running cassemdb process from cassemdb.pids"
	@ if [ -f "cassemdb.pids" ]; then \
		cat cassemdb.pids | xargs kill -9 || TRUE;\
	fi
	- rm cassemdb.pids
	#
	# If cassemdb process is not killed as expected, you can try following command:
	#
	# 1: kill -9 $$(ps -ef | grep cassemdb | awk '{print $2}')
	# 2: jobs -l | grep cassemdb | awk '{print $3}' | xargs kill -9

cassemdb.clear:
	- rm -fr ./cassemdb.pids
	- rm -fr ./debugdata/{1,2,3}/*

cassemadm.build:
	${GOCMD} build 	-o cassemadm \
					-ldflags "-s \
							  -X main.Version=`git tag --list | tail -n 1` \
							  -X main.BuildTime=`TZ=UTC date -u '+%Y-%m-%dT%H:%M:%SZ'` \
							  -X main.GitHash=`git rev-parse HEAD`" \
					./cmd/cassemadm

cassemadm.build-linux:
	${GOCMD_LINUX} build 	-o cassemadm \
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

cassemagent.linux-build:
	${GOCMD_LINUX} 	-o cassemagent \
					-ldflags "-s \
							  -X main.Version=`git tag --list | tail -n 1` \
							  -X main.BuildTime=`TZ=UTC date -u '+%Y-%m-%dT%H:%M:%SZ'` \
							  -X main.GitHash=`git rev-parse HEAD`" \
					./cmd/cassemagent

cassemagent.run: cassemagent.build
	DEBUG=1 ./cassemagent --conf=./examples/cassemagent/cassemagent.toml

build-all: cassemadm.build cassemagent.build cassemdb.build

cassemdb.image: cassemdb.build-linux
	docker build -t yeqown/cassemdb:${IMAGE_TAG} -f ./.deploy/dockerfiles/cassemdb.Dockerfile .
	docker push yeqown/cassemdb:${IMAGE_TAG}

cassemadm.image: cassemadm.build-linux
	docker build -t yeqown/cassemadm:${IMAGE_TAG} -f ./.deploy/dockerfiles/cassemadm.Dockerfile .
	docker push yeqown/cassemadm:${IMAGE_TAG}


cassemagent.image: cassemadm.build-linux
	docker build -t yeqown/cassemagent:${IMAGE_TAG} -f ./.deploy/dockerfiles/cassemagent.Dockerfile .
	docker push yeqown/cassemagent:${IMAGE_TAG}

image-all: cassemdb.image cassemadm.image cassemagent.image

proto-all:
	make -C ./internal/cassemdb/api
	make -C ./internal/concept
	make -C ./internal/cassemagent/api

clear:
	- rm ./cassemdb || rm ./cassemadm || rm ./cassemagent