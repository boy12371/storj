// Copyright (C) 2018 Storj Labs, Inc.
// See LICENSE for copying information.

package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"path"
	"regexp"

	_ "github.com/mattn/go-sqlite3"
	"google.golang.org/grpc"

	"storj.io/storj/pkg/piecestore/rpc/server/api"
	"storj.io/storj/pkg/piecestore/rpc/server/ttl"
	pb "storj.io/storj/protos/piecestore"
)

func main() {
	port := "7777"

	if len(os.Args) > 1 {
		if matched, _ := regexp.MatchString(`^\d{2,6}$`, os.Args[1]); matched == true {
			port = os.Args[1]
		}
	}

	dataDir := path.Join("./piece-store-data/", port)

	ttlDB, err := ttl.NewTTL("ttl-data.db")
	if err != nil {
		log.Fatalf("failed to open DB")
	}

	// create a listener on TCP port
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	// create a server instance
	s := api.Server{PieceStoreDir: dataDir, DB: ttlDB}

	// create a gRPC server object
	grpcServer := grpc.NewServer()

	// attach the api service to the server
	pb.RegisterPieceStoreRoutesServer(grpcServer, &s)

	// routinely check DB for and delete expired entries
	go func() {
		err := s.DB.DBCleanup(dataDir)
		log.Printf("Error in DBCleanup: %v", err)
	}()

	// start the server
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %s", err)
	}
}