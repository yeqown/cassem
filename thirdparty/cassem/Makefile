.DEFAULT_GOAL := gen-proto

PROTOC ?= protoc
ROOT := ../..
MODULE := github.com/yeqown/cassem
INCLUDES := -I ..
CLIENT_PROTOS := cassem/client/types.proto cassem/client/agent.proto
INTERNAL_MODELS := cassem/internal/models.proto cassem/internal/acl.proto
INTERNAL_KV := cassem/internal/kv.proto cassem/internal/raft.proto
INTERNAL_DELIVERY := cassem/internal/delivery.proto

.PHONY: gen-proto gen-internal gen-sdk-go

gen-proto: gen-sdk-go gen-internal

gen-sdk-go:
	$(PROTOC) $(INCLUDES) --go_out=module=$(MODULE):$(ROOT) --go-grpc_out=module=$(MODULE):$(ROOT) $(CLIENT_PROTOS)

gen-internal:
	$(PROTOC) $(INCLUDES) --go_out=module=$(MODULE):$(ROOT) $(INTERNAL_MODELS)
	$(PROTOC) $(INCLUDES) --go_out=module=$(MODULE),plugins=grpc:$(ROOT) $(INTERNAL_KV)
	$(PROTOC) $(INCLUDES) --go_out=module=$(MODULE):$(ROOT) --go-grpc_out=module=$(MODULE):$(ROOT) $(INTERNAL_DELIVERY)

