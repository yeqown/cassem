gen-proto:
	protoc \
		-I. \
		-I../../thirdparty \
		--go_out=plugins=grpc:.\
		--validate_out=lang=go:. \
		types.proto acl.proto