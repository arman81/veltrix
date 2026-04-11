package rules

import "log"

type Metric struct {
	GPUId       string  `json:"gpu_id"`
	Utilization float64 `json:"utilization"`
	Memory      float64 `json:"memory"`
}

type Recommendation struct {
	GPUId  string
	Action string
	Reason string
}

func Evaluate(m Metric) *Recommendation {

	// Rule 1: Underutilized GPU
	if m.Utilization < 20 && m.Memory < 40 {
		return &Recommendation{
			GPUId:  m.GPUId,
			Action: "SCALE_DOWN",
			Reason: "Low utilization detected",
		}
	}

	// Rule 2: Memory pressure
	if m.Memory > 90 {
		return &Recommendation{
			GPUId:  m.GPUId,
			Action: "INCREASE_MEMORY",
			Reason: "High memory usage",
		}
	}

	return nil
}

func Print(rec *Recommendation) {
	if rec != nil {
		log.Printf("RECOMMENDATION → GPU: %s | Action: %s | Reason: %s\n",
			rec.GPUId, rec.Action, rec.Reason)
	}
}
