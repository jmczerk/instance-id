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

func invoke(presigned *v4.PresignedHTTPRequest) {
	req := &http.Request{
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

	for key, vals := range rsp.Header {
		for _, val := range vals {
			log.Printf("HEADER %v : %v", key, val)
		}
	}

	log.Printf("BODY:\n%v", string(body))
}

func main() {
	ctx := context.Background()
	auth, err := utils.AuthenticateInstance(ctx)

	if err != nil {
		log.Fatalf("AuthenticateInstance failed: %v", err)
	}

	auth.SignedHeader.Set("X-Inverting-Proxy-EC2-VM-Desc", "eyJSZWdpb24iOiJ1cy1lYXN0LTEiLCJJbnN0YW5jZUlkIjoiaS0wZWJhM2ZlYjMwMGFkNWM2MiIsIlRhZ3MiOnsiQ2xpU2VydmVyTmFtZSI6InZlcmlseS1kZXZlbCIsIkVudmlyb25tZW50IjoiZGV2ZWwiLCJOYW1lIjoiZWMyaW5zdGFuY2UyMzEwMjQyMTAwIiwiUmVzb3VyY2VJZCI6IjQ2NDcyMDdkLTM1OGUtNGExOC1hMzJkLTA2NzZkMWM2NmJjMSIsIlRlbmFudCI6InNhYXMiLCJVc2VySUQiOiIyNjMxNzQwNjg3NDkxMTgxYzFkOTUiLCJWZXJzaW9uIjoidjAiLCJXb3Jrc3BhY2VJZCI6ImJkNDk4YTc2LTk0ZGYtNDM2NS1iMTM5LTRkNmRmZDExNDc2MCJ9fQ==")
	invoke(auth)
}
