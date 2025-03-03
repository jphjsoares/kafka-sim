package main

import (
	"log"
	"os"

	"github.com/jphjsoares/kafka-sim/pkg/kafka"
)


func main() {
	kafkaBrokerHost := os.Getenv("KAFKA_BROKER_NAME")
	kafkaBrokerPort := os.Getenv("KAFKA_BROKER_PORT")
	topic := os.Getenv("KAFKA_TOPIC")
	dlqTopic := os.Getenv("KAFKA_DLQ_TOPIC")

	if kafkaBrokerHost == "" || kafkaBrokerPort == "" || topic == "" || dlqTopic == "" {
		panic("KAFKA_BROKER_HOST, KAFKA_BROKER_PORT, KAFKA_TOPIC and KAFKA_DLQ_TOPIC must be set")
	}

	consumer, err := kafka.NewKafkaConsumer(
		[]string{kafkaBrokerHost + ":" + kafkaBrokerPort},
		topic, 
		dlqTopic,
	)
	if err != nil {
		log.Fatalf("Failed to create consumer: %v", err)
	}

	consumer.Consume()
}