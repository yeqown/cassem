gen-proto:
	protoc \
		-I. \
		-I../../thirdparty \
		--go_out=paths=source_relative,plugins=grpc:.\
		--validate_out=paths=source_relative,lang=go:. \
		types.proto acl.proto