package main

import (
	"context"
	"flag"
	"io"
	"log"
	"net/http"
	"net/url"

	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/jmczerk/instance-id/utils"
)

var (
	threads    = flag.Int("threads", 1, "Number of threads")
	iterations = flag.Int("iterations", 1, "Number of times to run per thread")
	sleepTime  = flag.Int("max-sleep-time", 10, "Max time to sleep between iterations")
)

func invoke(presigned *v4.PresignedHTTPRequest) string {
	req := http.Request{
		Method: presigned.Method,
		URL: func() *url.URL {
			parse, err := url.Parse(presigned.URL)
			if err != nil {
				panic(err)
			}
			return parse
		}(),
		Header: presigned.SignedHeader,
	}

	rsp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalf("Invoke HTTP request failed: %v", err)
	}

	body, err := io.ReadAll(rsp.Body)
	if err != nil {
		log.Fatalf("Read body failed: %v", err)
	}

	return string(body)
}

func main() {
	ctx := context.Background()
	auth, err := utils.AuthenticateInstance(ctx)

	if err != nil {
		log.Fatalf("AuthenticateInstance failed: %v", err)
	}

	log.Println(invoke(auth))
}
