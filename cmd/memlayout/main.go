package main

import (
	"flag"
	"fmt"
	"net"
	"os"

	"github.com/mloncode/memlayout"
	"google.golang.org/grpc"
	"gopkg.in/src-d/go-log.v1"
	lookout "gopkg.in/src-d/lookout-sdk.v0/pb"
)

const (
	version           = "dev"
	defaultPort       = 3456
	defaultDataServer = "localhost:10301"
	maxMessageSize    = 100 * 1024 * 1024 // 100mb
)

func main() {
	var port uint
	var dataServer string

	flag.UintVar(&port, "port", defaultPort, "port the server will bind to")
	flag.StringVar(&dataServer, "data-server", defaultDataServer, "address of the lookout data server")
	flag.Parse()

	l, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", port))
	if err != nil {
		log.Errorf(err, "failed to listen on port: %d", port)
		os.Exit(1)
	}

	opts := []grpc.ServerOption{
		grpc.MaxRecvMsgSize(maxMessageSize),
		grpc.MaxSendMsgSize(maxMessageSize),
	}

	s := grpc.NewServer(opts...)
	lookout.RegisterAnalyzerServer(s, memlayout.NewAnalyzer(version, dataServer))
	log.Infof("starting gRPC Analyzer server at port %d", port)
	s.Serve(l)
}
