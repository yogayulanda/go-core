package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/yogayulanda/go-core/httpclient"
	"github.com/yogayulanda/go-core/logger"
)

func main() {
	// 1. Initialize Logger
	// In a real go-core service, this is usually part of the application bootstrap
	l, err := logger.New("example-service", "debug")
	if err != nil {
		log.Fatalf("failed to initialize logger: %v", err)
	}

	// 2. Initialize Resilient HTTP Client
	// NewClient takes a logger for structured ServiceLog emission
	client := httpclient.NewClient(l,
		httpclient.WithTimeout(5*time.Second),
		httpclient.WithUserAgent("go-core-example/1.0"),
	)

	ctx := context.Background()

	// 3. Simple GET request
	fmt.Println("--- Performing simple GET request ---")
	resp, err := client.Get(ctx, "https://httpbin.org/get")
	if err != nil {
		fmt.Printf("GET request failed: %v\n", err)
	} else {
		fmt.Printf("GET response status: %d\n", resp.StatusCode())
	}

	// 4. POST request with custom payload
	fmt.Println("\n--- Performing POST request with body ---")
	payload := map[string]string{"task": "document everything"}
	resp, err = client.Post(ctx, "https://httpbin.org/post", payload)
	if err != nil {
		fmt.Printf("POST request failed: %v\n", err)
	} else {
		fmt.Printf("POST response status: %d\n", resp.StatusCode())
	}

	// 5. Using the Request() builder for more control
	fmt.Println("\n--- Using Request builder for custom headers ---")
	req := client.Request().SetHeader("X-Custom-Header", "go-core-value")
	resp, err = client.Do(ctx, req, http.MethodGet, "https://httpbin.org/headers")
	if err != nil {
		fmt.Printf("Custom request failed: %v\n", err)
	} else {
		fmt.Printf("Custom request status: %d\n", resp.StatusCode())
	}
}
