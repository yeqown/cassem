all: gen-proto gen-proto2

gen-proto:
	protoc \
		-I. \
		-I ../../../thirdparty \
		--go_out=plugins=grpc:. \
		--validate_out=lang=go:. \
		cassemdb.api.proto

gen-proto2:
	protoc \
		-I. \
		-I ../../../thirdparty \
		--go_out=plugins=grpc:. \
		--validate_out=lang=go:. \
		cassemdb.raft.proto
