module example

go 1.15

require (
	github.com/yeqown/cassem/clientv1 v0.0.0-20210301100658-a79bdb2e9a7a
)

replace (
	github.com/yeqown/cassem/clientv1 v0.0.0-20210301100658-a79bdb2e9a7a => ../client/v1
)
