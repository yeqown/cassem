package main

import (
	"fmt"

	clientv1 "github.com/yeqown/cassem/clientv1"
)

func main() {
	client, err := clientv1.New(&clientv1.Config{
		Endpoint: "127.0.0.1:2021",
		Watching: []clientv1.WatchContainerOption{
			{
				Namespace: "ns",
				Keys:      []string{"del-container-test"},
				Format:    "json",
			},
		},
		Fn: func(c clientv1.Changes) {
			fmt.Printf("changes trigger: %+v\n", c)
		},
	})

	if err != nil {
		panic(err)
	}
	_ = client
	// block to wait signal
	select {}
}
