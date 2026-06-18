all: gen-proto

gen-proto:
	DEBUG_PGV=1 protoc \
		-I. \
		-I ../../../thirdparty \
		--go_out=paths=source_relative,plugins=grpc:. \
		--validate_out=paths=source_relative,lang=go:. \
		cassemdb.api.proto cassemdb.raft.proto

#gen-validate:
#	DEBUG_PGV=1 protoc \
#		-I. \
#		-I ../../../thirdparty \
#		--validate_out=paths=source_relative,lang=go:. \
#		cassemdb.api.proto cassemdb.raft.proto