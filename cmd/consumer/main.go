package main

import (
	"os"

	"github.com/jphjsoares/kafka-sim/pkg/kafka"
)

func main() {
	kafkaBrokerHost := os.Getenv("KAFKA_BROKER_INTERNAL_ADDR")
	topic := os.Getenv("KAFKA_TOPIC")
	dlqTopic := os.Getenv("KAFKA_DLQ_TOPIC")

	if kafkaBrokerHost == "" || topic == "" || dlqTopic == "" {
		panic("KAFKA_BROKER_INTERNAL_ADDR, KAFKA_TOPIC, KAFKA_DLQ_TOPIC must be set")
	}

	// Initialize the Kafka consumer
	consumer := kafka.NewKafkaConsumer(
		[]string{kafkaBrokerHost},
		topic,
		dlqTopic,
	)

	// Start consuming messages
	consumer.Consume()
}