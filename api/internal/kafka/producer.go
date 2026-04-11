package kafka

import (
	"log"

	"github.com/IBM/sarama"
)

type Producer struct {
	client sarama.SyncProducer
}

func NewProducer(broker string) *Producer {
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true

	producer, err := sarama.NewSyncProducer([]string{broker}, config)
	if err != nil {
		log.Fatal(err)
	}

	return &Producer{client: producer}
}

func (p *Producer) Send(topic string, msg []byte) {
	_, _, err := p.client.SendMessage(&sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.ByteEncoder(msg),
	})

	if err != nil {
		log.Println("Kafka error:", err)
	}
}
