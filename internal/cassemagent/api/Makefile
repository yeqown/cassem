gen-proto:
	protoc \
		-I . \
		-I ../../ \
		-I ../../../thirdparty \
		--go_opt=Mconcept/types.proto=github.com/yeqown/cassem/concept \
		--go_out=plugins=grpc:. \
		--validate_out=lang=go:. \
		cassemagent.api.proto