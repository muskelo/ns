package main

import (
	"github.com/caarlos0/env/v8"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/muskelo/ns_server/httpadapter/internal/server"
	pb "github.com/muskelo/ns_server/protos/storage"
)

type config struct {
	StorageAddr string `env:"NS_HTTPADAPTER_STORAGE_ADDR" envDefault:"storage:5200"`
	Listen     string `env:"NS_HTTPADAPTER_LISTEN" envDefault:"0.0.0.0:5300"`
}

func main() {
	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		panic(err)
	}

	conn, err := grpc.Dial(cfg.StorageAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	client := pb.NewStorageServiceClient(conn)

	err = server.Run(client, cfg.Listen)
	if err != nil {
		panic(err)
	}
}
