package main

import (
	"context"
	"fmt"
	"time"

	"github.com/yeqown/cassem/api/agent"
	"github.com/yeqown/cassem/api/concept"
)

func main() {
	c, err := agent.New("127.0.0.1:20219",
		agent.WithClientId("clientId"), agent.WithClientIp("127.0.0.1"))
	if err != nil {
		panic(err)
	}

	_ = c.Watch(context.Background(),
		"test",
		"default",
		func(next *concept.Element) {
			fmt.Println("client one change: ", next.Metadata.Key, next.Raw)
		},
		"ele1", "bench02",
	)

	// query 4 times
	for i := 0; i < 4; i++ {
		time.Sleep(3 * time.Second)
		elems, err := c.GetElement(context.Background(), "test", "default", "ele1")
		if err != nil {
			panic(err)
		}

		fmt.Printf("element: %+v\n", elems)
	}

	// blocked here
	select {}
}
