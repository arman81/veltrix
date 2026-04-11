package kafka

import (
	"log"

	"github.com/IBM/sarama"
)

type Consumer struct {
	client sarama.Consumer
}

func NewConsumer(broker string) *Consumer {
	client, err := sarama.NewConsumer([]string{broker}, nil)
	if err != nil {
		log.Fatal(err)
	}

	return &Consumer{client: client}
}

func (c *Consumer) Consume(topic string, handler func([]byte)) {

	partitionConsumer, err := c.client.ConsumePartition(topic, 0, sarama.OffsetNewest)
	if err != nil {
		log.Fatal(err)
	}

	for msg := range partitionConsumer.Messages() {
		handler(msg.Value)
	}
}
