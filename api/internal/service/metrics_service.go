package service

import (
	"database/sql"
	"encoding/json"
	"log"

	"veltrix/api/internal/kafka"
	pb "veltrix/proto/gen"
)

type MetricsService struct {
	DB       *sql.DB
	Producer *kafka.Producer
}

func (s *MetricsService) Handle(batch *pb.MetricBatch) {

	for _, m := range batch.Metrics {

		// 1. Store in Postgres
		_, err := s.DB.Exec(
			"INSERT INTO metrics (gpu_id, utilization, memory) VALUES ($1, $2, $3)",
			m.GpuId, m.Utilization, m.Memory,
		)

		if err != nil {
			log.Println("DB error:", err)
		}

		// 2. Send to Kafka
		data, _ := json.Marshal(m)
		s.Producer.Send("metrics", data)
	}
}
