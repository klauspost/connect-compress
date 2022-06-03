// Copyright 2022 Klaus Post.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package compress_test

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"

	"github.com/bufbuild/connect-go"
	compress "github.com/klauspost/connect-compress"
	pingv1 "github.com/klauspost/connect-compress/internal/gen/connect/ping/v1"
	"github.com/klauspost/connect-compress/internal/gen/connect/ping/v1/pingv1connect"
)

func ExampleAll() {
	// Get client and server options for all compressors...
	clientOpts, serverOpts := compress.All(compress.LevelBalanced)

	// Create a server.
	_, h := pingv1connect.NewPingServiceHandler(&pingServer{}, serverOpts)
	srv := httptest.NewServer(h)
	client := pingv1connect.NewPingServiceClient(
		http.DefaultClient,
		srv.URL,
		clientOpts,
		// Compress requests with S2.
		connect.WithSendCompression(compress.S2),
	)
	req := connect.NewRequest(&pingv1.PingRequest{
		Number: 42,
	})
	req.Header().Set("Some-Header", "hello from connect")
	res, err := client.Ping(context.Background(), req)
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Println("The answer is", res.Msg)
	fmt.Println(res.Header().Get("Some-Other-Header"))
	//OUTPUT:
	//hello from connect
	//The answer is number:42
	//hello!
}

func ExampleSelect() {
	// Add Zstandard.
	clientOpts, serverOpts := compress.Select(compress.Zstandard, compress.LevelBalanced)
	_, h := pingv1connect.NewPingServiceHandler(&pingServer{}, serverOpts)
	srv := httptest.NewServer(h)
	client := pingv1connect.NewPingServiceClient(
		http.DefaultClient,
		srv.URL,
		clientOpts,
		// Enable request compression
		connect.WithSendCompression(compress.Zstandard),
	)
	req := connect.NewRequest(&pingv1.PingRequest{
		Number: 42,
	})
	req.Header().Set("Some-Header", "hello from connect")
	res, err := client.Ping(context.Background(), req)
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Println("The answer is", res.Msg)
	fmt.Println(res.Header().Get("Some-Other-Header"))
	//OUTPUT:
	//hello from connect
	//The answer is number:42
	//hello!
}

func ExampleSelect2() {
	// Add Zstandard.
	clientOpts, serverOpts := compress.Select(compress.Snappy, compress.LevelBalanced)
	_, h := pingv1connect.NewPingServiceHandler(&pingServer{}, serverOpts)
	srv := httptest.NewServer(h)
	client := pingv1connect.NewPingServiceClient(
		http.DefaultClient,
		srv.URL,
		clientOpts,
		// Enable request compression
		connect.WithSendCompression(compress.Snappy),
	)
	req := connect.NewRequest(&pingv1.PingRequest{
		Number: 42,
	})
	req.Header().Set("Some-Header", "hello from connect")
	res, err := client.Ping(context.Background(), req)
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Println("The answer is", res.Msg)
	fmt.Println(res.Header().Get("Some-Other-Header"))
	//OUTPUT:
	//hello from connect
	//The answer is number:42
	//hello!
}

func ExampleSelect3() {
	// Add Zstandard.
	clientOpts, serverOpts := compress.Select(compress.S2, compress.LevelBalanced)
	_, h := pingv1connect.NewPingServiceHandler(&pingServer{}, serverOpts)
	srv := httptest.NewServer(h)
	client := pingv1connect.NewPingServiceClient(
		http.DefaultClient,
		srv.URL,
		clientOpts,
		// Enable request compression
		connect.WithSendCompression(compress.S2),
	)
	req := connect.NewRequest(&pingv1.PingRequest{
		Number: 42,
	})
	req.Header().Set("Some-Header", "hello from connect")
	res, err := client.Ping(context.Background(), req)
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Println("The answer is", res.Msg)
	fmt.Println(res.Header().Get("Some-Other-Header"))
	//OUTPUT:
	//hello from connect
	//The answer is number:42
	//hello!
}

func ExampleSelect4() {
	// Add Zstandard.
	clientOpts, serverOpts := compress.Select(compress.Gzip, compress.LevelBalanced)
	_, h := pingv1connect.NewPingServiceHandler(&pingServer{}, serverOpts)
	srv := httptest.NewServer(h)
	client := pingv1connect.NewPingServiceClient(
		http.DefaultClient,
		srv.URL,
		clientOpts,
		// Enable request compression
		connect.WithSendCompression(compress.Gzip),
	)
	req := connect.NewRequest(&pingv1.PingRequest{
		Number: 42,
	})
	req.Header().Set("Some-Header", "hello from connect")
	res, err := client.Ping(context.Background(), req)
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Println("The answer is", res.Msg)
	fmt.Println(res.Header().Get("Some-Other-Header"))
	//OUTPUT:
	//hello from connect
	//The answer is number:42
	//hello!
}

type pingServer struct {
	pingv1connect.UnimplementedPingServiceHandler // returns errors from all methods
}

func (ps *pingServer) Ping(
	ctx context.Context,
	req *connect.Request[pingv1.PingRequest],
) (*connect.Response[pingv1.PingResponse], error) {
	// connect.Request and connect.Response give you direct access to headers and
	// trailers. No context-based nonsense!
	fmt.Println(req.Header().Get("Some-Header"))
	res := connect.NewResponse(&pingv1.PingResponse{
		// req.Msg is a strongly-typed *pingv1.PingRequest, so we can access its
		// fields without type assertions.
		Number: req.Msg.Number,
	})
	res.Header().Set("Some-Other-Header", "hello!")
	return res, nil
}
