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
	"strings"

	"connectrpc.com/connect/v2"
	"connectrpc.com/connect/v2/connecthttp"
	"github.com/klauspost/connect-compress/v3"
	pingv1 "github.com/klauspost/connect-compress/v3/internal/gen/connect/ping/v1"
	"github.com/klauspost/connect-compress/v3/internal/gen/connect/ping/v1/pingv1connect"
)

func ExampleWithAll() {
	// Get the client and server option for all compressors...
	opts := compress.WithAll(compress.LevelBalanced)

	// Create a server.
	server := connect.NewServer()
	pingv1connect.RegisterPingServiceHandler(server, &pingServer{})
	mux := http.NewServeMux()
	connecthttp.Mount(mux, server, opts)
	srv := httptest.NewServer(mux)
	client := pingv1connect.NewPingServiceClient(connect.NewClient(
		connecthttp.NewTransport(
			http.DefaultClient,
			srv.URL,
			opts,
			// Compress requests with S2.
			connecthttp.WithSendCompression(compress.S2),
		),
	))
	ctx, info := connect.NewClientContext(context.Background())
	info.RequestHeader().Set("Some-Header", "hello from connect")
	res, err := client.Ping(ctx, &pingv1.PingRequest{
		Number: 42,
	})
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Println("The answer is", res)
	fmt.Println(info.ResponseHeader().Get("Some-Other-Header"))
	//OUTPUT:
	//hello from connect
	//The answer is number:42
	//hello!
}

func ExampleNew_zstd() {
	// Add Zstandard.
	opt := connecthttp.WithCompressors(compress.New(compress.Zstandard, compress.LevelBalanced))
	server := connect.NewServer()
	pingv1connect.RegisterPingServiceHandler(server, &pingServer{})
	mux := http.NewServeMux()
	connecthttp.Mount(mux, server, opt)
	srv := httptest.NewServer(mux)
	client := pingv1connect.NewPingServiceClient(connect.NewClient(
		connecthttp.NewTransport(
			http.DefaultClient,
			srv.URL,
			opt,
			// Enable request compression
			connecthttp.WithSendCompression(compress.Zstandard),
		),
	))
	ctx, info := connect.NewClientContext(context.Background())
	info.RequestHeader().Set("Some-Header", "hello from connect")
	res, err := client.Ping(ctx, &pingv1.PingRequest{
		Number: 42,
	})
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Println("The answer is", res)
	fmt.Println(info.ResponseHeader().Get("Some-Other-Header"))
	//OUTPUT:
	//hello from connect
	//The answer is number:42
	//hello!
}

func ExampleNew_snappy() {
	// Add Zstandard.
	opt := connecthttp.WithCompressors(compress.New(compress.Snappy, compress.LevelBalanced))
	server := connect.NewServer()
	pingv1connect.RegisterPingServiceHandler(server, &pingServer{})
	mux := http.NewServeMux()
	connecthttp.Mount(mux, server, opt)
	srv := httptest.NewServer(mux)
	client := pingv1connect.NewPingServiceClient(connect.NewClient(
		connecthttp.NewTransport(
			http.DefaultClient,
			srv.URL,
			opt,
			// Enable request compression
			connecthttp.WithSendCompression(compress.Snappy),
		),
	))
	ctx, info := connect.NewClientContext(context.Background())
	info.RequestHeader().Set("Some-Header", "hello from connect")
	res, err := client.Ping(ctx, &pingv1.PingRequest{
		Number: 42,
	})
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Println("The answer is", res)
	fmt.Println(info.ResponseHeader().Get("Some-Other-Header"))
	//OUTPUT:
	//hello from connect
	//The answer is number:42
	//hello!
}

func ExampleNew_s2() {
	// Add Zstandard.
	opt := connecthttp.WithCompressors(compress.New(compress.S2, compress.LevelBalanced))
	server := connect.NewServer()
	pingv1connect.RegisterPingServiceHandler(server, &pingServer{})
	mux := http.NewServeMux()
	connecthttp.Mount(mux, server, opt)
	srv := httptest.NewServer(mux)
	client := pingv1connect.NewPingServiceClient(connect.NewClient(
		connecthttp.NewTransport(
			http.DefaultClient,
			srv.URL,
			opt,
			// Enable request compression
			connecthttp.WithSendCompression(compress.S2),
		),
	))
	ctx, info := connect.NewClientContext(context.Background())
	info.RequestHeader().Set("Some-Header", "hello from connect")
	res, err := client.Ping(ctx, &pingv1.PingRequest{
		Number: 42,
	})
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Println("The answer is", res)
	fmt.Println(info.ResponseHeader().Get("Some-Other-Header"))
	//OUTPUT:
	//hello from connect
	//The answer is number:42
	//hello!
}

func ExampleNew_gzip() {
	// Add Zstandard.
	opt := connecthttp.WithCompressors(compress.New(compress.Gzip, compress.LevelBalanced))
	server := connect.NewServer()
	pingv1connect.RegisterPingServiceHandler(server, &pingServer{})
	mux := http.NewServeMux()
	connecthttp.Mount(mux, server, opt)
	srv := httptest.NewServer(mux)
	client := pingv1connect.NewPingServiceClient(connect.NewClient(
		connecthttp.NewTransport(
			http.DefaultClient,
			srv.URL,
			opt,
			// Enable request compression
			connecthttp.WithSendCompression(compress.Gzip),
		),
	))
	ctx, info := connect.NewClientContext(context.Background())
	info.RequestHeader().Set("Some-Header", "hello from connect")
	res, err := client.Ping(ctx, &pingv1.PingRequest{
		Number: 42,
	})
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Println("The answer is", res)
	fmt.Println(info.ResponseHeader().Get("Some-Other-Header"))
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
	req *pingv1.PingRequest,
) (*pingv1.PingResponse, error) {
	// connect.CallInfo gives you access to headers and trailers.
	info, _ := connect.CallInfoForServerContext(ctx)
	fmt.Println(info.RequestHeader().Get("Some-Header"))
	res := &pingv1.PingResponse{
		// req is a strongly-typed *pingv1.PingRequest, so we can access its
		// fields without type assertions.
		Number: req.Number,
		Text:   req.Text,
	}
	info.ResponseHeader().Set("Some-Other-Header", "hello!")
	return res, nil
}

func ExampleNew_minlz() {
	// Use MinLZ.
	opt := connecthttp.WithCompressors(compress.New(compress.MinLZ, compress.LevelBalanced))
	server := connect.NewServer()
	pingv1connect.RegisterPingServiceHandler(server, &pingServer{})
	mux := http.NewServeMux()
	connecthttp.Mount(mux, server, opt)
	srv := httptest.NewServer(mux)
	client := pingv1connect.NewPingServiceClient(connect.NewClient(
		connecthttp.NewTransport(
			http.DefaultClient,
			srv.URL,
			opt,
			// Enable request compression
			connecthttp.WithSendCompression(compress.MinLZ),
		),
	))
	ctx, info := connect.NewClientContext(context.Background())
	info.RequestHeader().Set("Some-Header", "hello from connect")
	res, err := client.Ping(ctx, &pingv1.PingRequest{
		Number: 42,
		Text:   strings.Repeat("a", 50),
	})
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Println("The answer is", res.Number)
	fmt.Println("Text is", res.Text)
	//OUTPUT:
	//hello from connect
	//The answer is 42
	//Text is aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
}
