GOCMD=CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go

build-cassemd:
	${GOCMD} build -o cassemd ./cmd/cassemd

build-cassemctl:
	${GOCMD} build -o cassemctl ./cmd/cassemctl

image: build-cassemd build-cassemctl
	docker build -t yeqown/cassem .

clear:
	@ rm ./cassemd || rm ./cassemctl