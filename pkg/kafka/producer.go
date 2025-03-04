package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/jphjsoares/kafka-sim/pkg/protobuf"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/proto"
)

type KafkaProducer struct {
	brokers        []string
	topic          string
	writer         *kafka.Writer
	validCounter   *prometheus.CounterVec
	invalidCounter *prometheus.CounterVec
}

func NewKafkaProducer(brokers []string, topic string) *KafkaProducer {
	validCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kafka_producer_valid_messages_total",
			Help: "Total number of valid messages produced by the Kafka producer",
		},
		[]string{"topic"}, // Label to categorize the metric by topic
	)
	invalidCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kafka_producer_invalid_messages_total",
			Help: "Total number of invalid messages produced by the Kafka producer",
		},
		[]string{"topic"}, // Label to categorize the metric by topic
	)

	prometheus.MustRegister(validCounter)
	prometheus.MustRegister(invalidCounter)

	producer := &KafkaProducer{
		brokers:        brokers,
		topic:          topic,
		writer: &kafka.Writer{
			Addr:     kafka.TCP(brokers...),
			Topic:    topic,
			Balancer: &kafka.LeastBytes{},
			BatchSize: 50, // Send messages in batches of 50
			BatchTimeout: 100 * time.Millisecond, // Wait up to 100ms to fill a batch
			Async: true, // Send messages asynchronously
		},
		validCounter:   validCounter,
		invalidCounter: invalidCounter,
	}

	go func() {
		http.Handle("/metrics", promhttp.Handler())
		log.Println("Prometheus metrics exposed at http://localhost:8081/metrics")
		log.Fatal(http.ListenAndServe(":8081", nil))
	}()

	return producer
}

func (kp *KafkaProducer) createMessage() *protobuf.Message {
	return &protobuf.Message{
		Id:        fmt.Sprintf("%d", rand.Intn(1000000)), // Generate a random ID (could be improved)
		Content:   "Hello World!",
		Timestamp: time.Now().Format(time.RFC3339),
	}
}

func (kp *KafkaProducer) GenerateMessages() {
	log.Println("Start sending message bursts...")
	for {
		log.Println("Sending batch...")
		messages := make([]kafka.Message, 0, 50)
		for i := 0; i < 50; i++ {
			var message proto.Message
			if rand.Float32() < 0.05 { // 5% chance of invalid message
				kp.invalidCounter.WithLabelValues(kp.topic).Inc()
				invalidMessage := map[string]string{"invalid_field": "Should be sent to DLQ!"}
				messageBytes, _ := json.Marshal(invalidMessage)
				messages = append(messages, kafka.Message{
					Value: messageBytes,
				})
			} else {
				kp.validCounter.WithLabelValues(kp.topic).Inc()
				message = kp.createMessage()
				messageBytes, _ := proto.Marshal(message)
				messages = append(messages, kafka.Message{
					Value: messageBytes,
				})
			}
		}
		// Write all messages in the batch at once
		err := kp.writer.WriteMessages(context.Background(), messages...)
		if err != nil {
			log.Printf("Failed to write messages: %v", err)
		} else {
			log.Printf("Produced batch of %d messages", len(messages))
		}

		time.Sleep(1 * time.Second)
	}
}

func (kp *KafkaProducer) Close() {
	kp.writer.Close()
}
