package main

import (
	"os"

	"github.com/jphjsoares/kafka-sim/pkg/kafka"
)

func main() {
	// Read environment variables
	kafkaBrokerHost := os.Getenv("KAFKA_BROKER_INTERNAL_ADDR")
	topic := os.Getenv("KAFKA_TOPIC")

	if kafkaBrokerHost == "" || topic == "" {
		panic("KAFKA_BROKER_INTERNAL_ADDR and KAFKA_TOPIC must be set")
	}

	// Initialize the Kafka producer
	producer := kafka.NewKafkaProducer(
		[]string{kafkaBrokerHost},
		topic,
	)

	// Start generating messages
	producer.GenerateMessages()
}