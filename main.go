package main

import (
	"context"
	"flag"
	"io"
	"log"
	"net/http"

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

	enc, err := auth.MarshalAndEncode()

	if err != nil {
		log.Fatalf("Encoding req failed: %v", err)
	}

	log.Printf("Encoded request:\n%v", enc)

	dec, err := utils.DecodeAndUnmarshal(enc)

	if err != nil {
		log.Fatalf("Decoding req failed: %v", err)
	}

	req := &http.Request{
		Method: dec.Method,
		URL:    &dec.URL,
		Header: dec.SignedHeader,
	}

	rsp, err := http.DefaultClient.Do(req)

	if err != nil {
		log.Fatalf("Perf req failed: %v", err)
	}

	body, err := io.ReadAll(rsp.Body)

	if err != nil {
		log.Fatalf("Reading body failed: %v", err)
	}

	log.Println(body)
}
