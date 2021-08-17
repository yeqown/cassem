package main

import (
	"context"
	"fmt"
	"time"

	apiagent "github.com/yeqown/cassem/internal/cassemagent/api"
	"github.com/yeqown/cassem/internal/concept"
)

func main() {
	c, err := apiagent.Dial("127.0.0.1:20219")
	if err != nil {
		panic(err)
	}
	go func() {
		err = c.Wait(context.Background(),
			"app",
			"env",
			"clientId",
			"127.0.0.1",
			func(next *concept.Element) {
				fmt.Println("client one change: ", next.Metadata.Key, next.Raw)
			},
			"ele1", "bench02",
		)
		if err != nil {
			panic(err)
		}
	}()

	// query 4 times
	for i := 0; i < 4; i++ {
		time.Sleep(3 * time.Second)
		elems, err := c.GetConfig(context.Background(), "app", "env", "ele1")
		if err != nil {
			panic(err)
		}

		fmt.Printf("element: %+v", elems)
	}

	// blocked here
	select {}
}
