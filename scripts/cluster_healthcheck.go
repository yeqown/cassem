//go:build ignore

package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/yeqown/cassem/scripts/clusterhealth"
)

func main() {
	endpoints := flag.String("endpoints", "127.0.0.1:2021,127.0.0.1:2022,127.0.0.1:2023", "comma-separated cassemdb endpoints")
	timeout := flag.Duration("timeout", 45*time.Second, "healthcheck timeout")
	flag.Parse()

	if err := clusterhealth.Check(strings.Split(*endpoints, ","), *timeout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
