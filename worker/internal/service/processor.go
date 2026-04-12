package service

import (
	"database/sql"
	"encoding/json"
	"log"

	"veltrix/worker/internal/rules"
)

type Processor struct {
	DB *sql.DB
}

func (p *Processor) ProcessMessage(msg []byte) {

	var metric rules.Metric

	err := json.Unmarshal(msg, &metric)
	if err != nil {
		log.Println("JSON error:", err)
		return
	}

	rec := rules.Evaluate(metric)

	if rec != nil {
		log.Println("Saving recommendation...")

		_, err := p.DB.Exec(
			"INSERT INTO recommendations (gpu_id, action, reason) VALUES ($1,$2,$3)",
			rec.GPUId, rec.Action, rec.Reason,
		)

		if err != nil {
			log.Println("DB error:", err)
		}
	}
}
