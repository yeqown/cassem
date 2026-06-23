.DEFAULT_GOAL := gen-proto

.PHONY: gen-proto concept.gen-proto kv.gen-proto agent.gen-proto

gen-proto: concept.gen-proto kv.gen-proto agent.gen-proto

concept.gen-proto:
	cd concept && protoc \
		-I . \
		-I ../../thirdparty \
		--go_out=paths=source_relative,plugins=grpc:. \
		--validate_out=paths=source_relative,lang=go:. \
		types.proto acl.proto

kv.gen-proto:
	cd kv && DEBUG_PGV=1 protoc \
		-I . \
		-I ../../thirdparty \
		--go_out=paths=source_relative,plugins=grpc:. \
		--validate_out=paths=source_relative,lang=go:. \
		cassemdb.api.proto cassemdb.raft.proto

agent.gen-proto:
	cd agent && protoc \
		-I . \
		-I ../ \
		-I ../../thirdparty \
		--go_opt=Mconcept/types.proto=github.com/yeqown/cassem/api/concept \
		--go_out=paths=source_relative:. \
		--go-grpc_out=paths=source_relative:. \
		--validate_out=paths=source_relative,lang=go:. \
		cassemagent.api.proto
