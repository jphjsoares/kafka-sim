package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"net/http"

	"github.com/jphjsoares/kafka-sim/pkg/protobuf"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/proto"
)

type KafkaConsumer struct {
	brokers       []string
	topic         string
	dlqTopic      string
	groupID       string
	consumer      *kafka.Reader
	dlqProducer   *kafka.Writer
	validCounter  *prometheus.CounterVec
	invalidCounter *prometheus.CounterVec
	running       bool
}

func NewKafkaConsumer(brokers []string, topic string, dlqTopic string) *KafkaConsumer {
	validCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kafka_consumer_valid_messages_total",
			Help: "Total number of valid messages consumed by the Kafka consumer",
		},
		[]string{"topic", "group_id"}, // Labels to categorize the metric
	)
	invalidCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kafka_consumer_invalid_messages_total",
			Help: "Total number of invalid messages consumed by the Kafka consumer",
		},
		[]string{"topic", "group_id"}, // Labels to categorize the metric
	)

	// Register the counters with Prometheus
	prometheus.MustRegister(validCounter)
	prometheus.MustRegister(invalidCounter)

	return &KafkaConsumer{
		brokers:       brokers,
		topic:         topic,
		dlqTopic:      dlqTopic,
		groupID:       "myconsumergroup",
		validCounter:  validCounter,
		invalidCounter: invalidCounter,
		running:       true,
	}
}

func (kc *KafkaConsumer) createConsumer() {
	kc.consumer = kafka.NewReader(kafka.ReaderConfig{
		Brokers:  kc.brokers,
		Topic:    kc.topic,
		GroupID:  kc.groupID,
	})

	kc.dlqProducer = &kafka.Writer{
		Addr:  kafka.TCP(kc.brokers...),
		Topic: kc.dlqTopic,
	}
}

func (kc *KafkaConsumer) shutdown() {
	log.Println("Shutdown received. Closing consumer...")
	kc.running = false
	kc.consumer.Close()
	kc.dlqProducer.Close()
}

func (kc *KafkaConsumer) sendToDLQ(msg kafka.Message, err error) {
	failedMessage := map[string]string{
		"original_message": string(msg.Value),
		"error":            err.Error(),
		"timestamp":        time.Now().Format(time.RFC3339),
	}
	failedMessageBytes, _ := json.Marshal(failedMessage)

	kc.dlqProducer.WriteMessages(context.Background(), kafka.Message{
		Value: failedMessageBytes,
	})
	kc.invalidCounter.WithLabelValues(kc.topic, kc.groupID).Inc()
	log.Printf("Sent to DLQ: %s", failedMessage)
}

func (kc *KafkaConsumer) processMessage(msg kafka.Message) {
	var message protobuf.Message
	if err := proto.Unmarshal(msg.Value, &message); err != nil {
		kc.sendToDLQ(msg, fmt.Errorf("failed to unmarshal message: %v", err))
		return
	}

	// Simulate message processing
	log.Printf("Consumed valid message: %v", message.Content)
	time.Sleep(50 * time.Millisecond) // Simulate processing delay

	kc.validCounter.WithLabelValues(kc.topic, kc.groupID).Inc()
}

func (kc *KafkaConsumer) Consume() {
	kc.createConsumer()
	defer kc.shutdown()

	// Handle graceful shutdown
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		http.Handle("/metrics", promhttp.Handler())
		log.Println("Prometheus metrics exposed at http://localhost:8081/metrics")
		log.Fatal(http.ListenAndServe(":8081", nil))
	}()

	log.Println("Consumer started and waiting for messages...")
	for kc.running {
		select {
		case <-sigchan:
			kc.shutdown()
		default:
			msg, err := kc.consumer.ReadMessage(context.Background())
			if err != nil {
				log.Printf("Failed to read message: %v", err)
				continue
			}
			kc.processMessage(msg)
		}
	}
}
