gen-proto:
	protoc \
		-I . \
		-I ../ \
		-I ../../thirdparty \
		--go_opt=Mconcept/types.proto=github.com/yeqown/cassem/api/concept \
		--go_out=paths=source_relative,plugins=grpc:. \
		--validate_out=paths=source_relative,lang=go:. \
		cassemagent.api.proto