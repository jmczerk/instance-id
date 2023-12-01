package main

import (
	"context"
	"flag"
	"log"

	"github.com/jmczerk/instance-id/utils"
)

var (
	threads    = flag.Int("threads", 1, "Number of threads")
	iterations = flag.Int("iterations", 1, "Number of times to run per thread")
	sleepTime  = flag.Int("max-sleep-time", 10, "Max time to sleep between iterations")
)

func main() {
	ctx := context.Background()
	auth, err := utils.AuthenticateInstance(ctx)

	if err != nil {
		log.Fatalf("AuthenticateInstance failed: %v", err)
	}

	log.Println(utils.MarshalAndEncode(auth))
}
