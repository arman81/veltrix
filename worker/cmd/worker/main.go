package main

import (
	"database/sql"
	"log"
	"os"

	"gpu-optimizer/worker/internal/kafka"
	"gpu-optimizer/worker/internal/service"
)

func main() {

	broker := os.Getenv("KAFKA_BROKER")
	consumer := kafka.NewConsumer(broker)
	db, _ := sql.Open("postgres", os.Getenv("POSTGRES_URL"))

	processor := &service.Processor{DB: db}

	consumer.Consume("metrics", processor.ProcessMessage)

	log.Println("Worker started, consuming Kafka...")

	consumer.Consume("metrics", service.ProcessMessage)

}
