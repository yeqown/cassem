all: gen-proto

gen-proto:
	DEBUG_PGV=1 protoc \
		-I. \
		-I ../../../thirdparty \
		--go_out=plugins=grpc:. \
		--validate_out=lang=go:. \
		cassemdb.api.proto cassemdb.raft.proto

