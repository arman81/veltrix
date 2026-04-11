package collector

import (
	"math/rand"
	"time"
)

type Metric struct {
	GPUId       string
	Utilization float32
	Memory      float32
	Timestamp   int64
}

func Collect() []Metric {

	return []Metric{
		{
			GPUId:       "gpu-0",
			Utilization: float32(rand.Intn(100)),
			Memory:      float32(rand.Intn(100)),
			Timestamp:   time.Now().Unix(),
		},
	}
}
