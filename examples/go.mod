module example

go 1.17


require (
	github.com/yeqown/cassem v0.2.0-rc2
	github.com/yeqown/cassem/api v1.0.0
)

replace (
	github.com/yeqown/cassem => ../
	github.com/yeqown/cassem/api => ../api
)

require (
	github.com/Knetic/govaluate v3.0.1-0.20171022003610-9aa49832a739+incompatible // indirect
	github.com/casbin/casbin/v2 v2.36.1 // indirect
	github.com/envoyproxy/protoc-gen-validate v0.6.1 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/yeqown/log v1.1.1 // indirect
	golang.org/x/crypto v0.0.0-20201002170205-7f63de1d35b0 // indirect
	golang.org/x/net v0.0.0-20210405180319-a5a99cb37ef4 // indirect
	golang.org/x/sys v0.0.0-20210603081109-ebe580a85c40 // indirect
	golang.org/x/text v0.3.5 // indirect
	google.golang.org/genproto v0.0.0-20210602131652-f16073e35f0c // indirect
	google.golang.org/grpc v1.40.0 // indirect
	google.golang.org/protobuf v1.27.1 // indirect
)
