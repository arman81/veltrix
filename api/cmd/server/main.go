package main

import (
	"log"
	"net"

	"gpu-optimizer/api/internal/config"
	"gpu-optimizer/api/internal/db"
	"gpu-optimizer/api/internal/grpc"
	"gpu-optimizer/api/internal/kafka"
	"gpu-optimizer/api/internal/service"

	pb "gpu-optimizer/proto/gen"

	"google.golang.org/grpc"
)

func main() {

	cfg := config.Load()

	// DB
	database := db.Connect(cfg.PostgresURL)

	// Kafka
	producer := kafka.NewProducer(cfg.KafkaBroker)

	// Service
	svc := &service.MetricsService{
		DB:       database,
		Producer: producer,
	}

	// gRPC server
	lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		log.Fatal(err)
	}

	grpcServer := grpc.NewServer()

	pb.RegisterMetricsServiceServer(grpcServer, &grpc.Server{
		Service: svc,
	})

	log.Println("API running on port", cfg.GRPCPort)

	grpcServer.Serve(lis)
}
