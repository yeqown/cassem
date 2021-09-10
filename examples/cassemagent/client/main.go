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
		agent.WithClientId("clientId2"), agent.WithClientIp("127.0.0.1"))
	if err != nil {
		panic(err)
	}

	_ = c.Watch(context.Background(),
		"test",
		"default",
		func(next *concept.Element) {
			fmt.Printf("ONE PUBLISH: %v, %s", next.Metadata.Key, next.Raw)
		},
		"ele1", "config",
	)

	// query 4 times
	for i := 0; i < 4; i++ {
		time.Sleep(3 * time.Second)
		elems, err := c.GetElement(context.Background(), "test", "default", "config")
		if err != nil {
			panic(err)
		}

		fmt.Printf("element: %+v\n", elems)
	}

	// blocked here
	select {}
}
