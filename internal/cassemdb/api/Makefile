gen-proto:
	protoc \
		-I. \
		-I ../../../thirdparty \
		--go_out=plugins=grpc:. \
		--validate_out=lang=go:. \
		api.proto