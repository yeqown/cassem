gen-proto:
	protoc \
		-I . \
		-I ../../ \
		-I ../../../thirdparty \
		--go_opt=Mconcept/types.proto=github.com/yeqown/cassem/internal/concept \
		--go_out=plugins=grpc:. \
		--validate_out=lang=go:. \
		api.proto