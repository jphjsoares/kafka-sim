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
		for range 50 {
			var message proto.Message
			if rand.Float32() < 0.05 { // 5% chance of invalid message
				kp.invalidCounter.WithLabelValues(kp.topic).Inc()
				invalidMessage := map[string]string{"invalid_field": "Should be sent to DLQ!"}
				messageBytes, _ := json.Marshal(invalidMessage)
				kp.writer.WriteMessages(context.Background(), kafka.Message{
					Value: messageBytes,
				})
				log.Printf("Produced invalid message: %s", invalidMessage)
			} else {
				kp.validCounter.WithLabelValues(kp.topic).Inc()
				message = kp.createMessage()
				messageBytes, _ := proto.Marshal(message)
				kp.writer.WriteMessages(context.Background(), kafka.Message{
					Value: messageBytes,
				})
				log.Printf("Produced valid message: %v", message)
			}
		}
		time.Sleep(1 * time.Second)
	}
}

func (kp *KafkaProducer) Close() {
	kp.writer.Close()
}
