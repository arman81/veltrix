package grpc

import (
	"context"
	"log"

	"gpu-optimizer/api/internal/service"
	pb "gpu-optimizer/proto/gen"
)

type Server struct {
	pb.UnimplementedMetricsServiceServer
	Service *service.MetricsService
}

func (s *Server) SendMetrics(ctx context.Context, req *pb.MetricBatch) (*pb.Ack, error) {

	log.Println("Received metrics from:", req.NodeId)

	s.Service.Handle(req)

	return &pb.Ack{Success: true}, nil
}
