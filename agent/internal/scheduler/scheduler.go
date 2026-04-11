package scheduler

import (
	"log"
	"time"

	"gpu-optimizer/agent/internal/collector"
	"gpu-optimizer/agent/internal/grpc"

	pb "gpu-optimizer/proto/gen"
)

func Start(endpoint string) {

	client := grpc.NewClient(endpoint)

	ticker := time.NewTicker(5 * time.Second)

	for {
		<-ticker.C

		raw := collector.Collect()

		var metrics []pb.Metric

		for _, m := range raw {
			metrics = append(metrics, pb.Metric{
				GpuId:       m.GPUId,
				Utilization: m.Utilization,
				Memory:      m.Memory,
				Timestamp:   m.Timestamp,
			})
		}

		log.Println("Sending metrics...")

		client.Send(metrics, "node-1")
	}
}
