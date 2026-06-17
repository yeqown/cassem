package main

import (
	"context"
	"flag"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/yeqown/cassem/api/agent"
	"github.com/yeqown/cassem/api/concept"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:20219", "cassemagent address")
	app := flag.String("app", "test", "app id")
	env := flag.String("env", "default", "env id")
	keyList := flag.String("keys", "ele1,config", "comma-separated watch keys")
	clientPrefix := flag.String("client-prefix", "client", "client id prefix")
	clientIPPrefix := flag.String("client-ip-prefix", "127.0.0.", "client ip prefix")
	count := flag.Int("count", 3, "client instance count")
	start := flag.Int("start", 1, "client id/ip start index")
	queryKey := flag.String("query-key", "config", "key used by periodic GetElement")
	queryInterval := flag.Duration("query-interval", 10*time.Second, "periodic GetElement interval")
	flag.Parse()

	keys := splitKeys(*keyList)
	if len(keys) == 0 {
		panic("keys could not be empty")
	}
	if *count <= 0 {
		panic("count must be greater than 0")
	}

	var wg sync.WaitGroup
	for i := 0; i < *count; i++ {
		idx := *start + i
		clientID := fmt.Sprintf("%s-%02d", *clientPrefix, idx)
		clientIP := fmt.Sprintf("%s%d", *clientIPPrefix, idx)
		wg.Add(1)
		go func() {
			defer wg.Done()
			runClient(*addr, clientID, clientIP, *app, *env, *queryKey, keys, *queryInterval)
		}()
	}
	wg.Wait()
}

func splitKeys(value string) []string {
	parts := strings.Split(value, ",")
	keys := make([]string, 0, len(parts))
	for _, part := range parts {
		key := strings.TrimSpace(part)
		if key != "" {
			keys = append(keys, key)
		}
	}
	return keys
}

func runClient(addr, clientID, clientIP, app, env, queryKey string, keys []string, queryInterval time.Duration) {
	c, err := agent.New(addr, agent.WithClientId(clientID), agent.WithClientIp(clientIP))
	if err != nil {
		panic(err)
	}
	defer c.Quit()

	fmt.Printf("client registered: id=%s ip=%s addr=%s app=%s env=%s keys=%v\n", clientID, clientIP, addr, app, env, keys)

	if err = c.Watch(context.Background(), app, env, func(next *concept.Element) {
		fmt.Printf("client=%s publish key=%s raw=%s\n", clientID, next.Metadata.Key, next.Raw)
	}, keys...); err != nil {
		panic(err)
	}

	ticker := time.NewTicker(queryInterval)
	defer ticker.Stop()
	for range ticker.C {
		elem, err := c.GetElement(context.Background(), app, env, queryKey)
		if err != nil {
			fmt.Printf("client=%s get key=%s err=%v\n", clientID, queryKey, err)
			continue
		}
		fmt.Printf("client=%s get key=%s elem=%+v\n", clientID, queryKey, elem)
	}
}
