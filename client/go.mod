module github.github.com/yeqown/cassem/client

go 1.17

require (
	github.com/pkg/errors v0.9.1
	github.com/yeqown/cassem v0.2.0-rc2
	github.com/yeqown/log v1.1.1
	google.golang.org/grpc v1.40.0
)

require (
	github.com/envoyproxy/protoc-gen-validate v0.6.1 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	golang.org/x/crypto v0.0.0-20201002170205-7f63de1d35b0 // indirect
	golang.org/x/net v0.0.0-20210405180319-a5a99cb37ef4 // indirect
	golang.org/x/sys v0.0.0-20210603081109-ebe580a85c40 // indirect
	golang.org/x/text v0.3.5 // indirect
	google.golang.org/genproto v0.0.0-20210602131652-f16073e35f0c // indirect
	google.golang.org/protobuf v1.27.1 // indirect
)

replace (
	github.com/yeqown/cassem => ../
)