package main

import (
	"os"

	"github.com/jphjsoares/kafka-sim/pkg/kafka"
)

func main() {
	// Read environment variables
	kafkaBrokerHost := os.Getenv("KAFKA_BROKER_NAME")
	kafkaBrokerPort := os.Getenv("KAFKA_BROKER_PORT")
	topic := os.Getenv("KAFKA_TOPIC")

	if kafkaBrokerHost == "" || kafkaBrokerPort == "" || topic == "" {
		panic("KAFKA_BROKER_HOST, KAFKA_BROKER_PORT, and KAFKA_TOPIC must be set")
	}

	// Initialize the Kafka producer
	producer := kafka.NewKafkaProducer(
		[]string{kafkaBrokerHost + ":" + kafkaBrokerPort},
		topic,
	)

	// Start generating messages
	producer.GenerateMessages()
}