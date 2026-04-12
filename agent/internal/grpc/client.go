package grpc

import (
	"context"
	"log"
	"time"

	pb "veltrix/proto/gen"

	"google.golang.org/grpc"
)

type Client struct {
	conn   *grpc.ClientConn
	client pb.MetricsServiceClient
}

func NewClient(endpoint string) *Client {

	conn, err := grpc.Dial(endpoint, grpc.WithInsecure())
	if err != nil {
		log.Fatal(err)
	}

	return &Client{
		conn:   conn,
		client: pb.NewMetricsServiceClient(conn),
	}
}

func (c *Client) Send(metrics []pb.Metric, nodeID string) {

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	batch := &pb.MetricBatch{
		NodeId:  nodeID,
		Metrics: metrics,
	}

	_, err := c.client.SendMetrics(ctx, batch)
	if err != nil {
		log.Println("gRPC error:", err)
	}
}
