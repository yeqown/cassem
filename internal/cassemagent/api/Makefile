gen-proto:
	protoc -I ../../ \
	-I . \
	--go_opt=Mconcept/types.proto=github.com/yeqown/cassem/internal/concept \
	--go_out=plugins=grpc:. \
	api.proto