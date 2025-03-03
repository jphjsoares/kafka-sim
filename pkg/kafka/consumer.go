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

	"github.com/jphjsoares/kafka-sim/pkg/protobuf"
	"github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/proto"
)

type KafkaConsumer struct {
	brokers      []string
	topic        string
	dlqTopic     string
	groupID      string
	consumer     *kafka.Reader
	dlqProducer  *kafka.Writer
	validCounter int
	invalidCounter int
	running      bool
}

func NewKafkaConsumer(brokers []string, topic string, dlqTopic string) *KafkaConsumer {
	return &KafkaConsumer{
		brokers:  brokers,
		topic:    topic,
		dlqTopic: dlqTopic,
		groupID:  "myconsumergroup",
		running:  true,
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
	kc.invalidCounter++
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

	kc.validCounter++
	log.Printf("Consumed messages: %d", kc.validCounter)
	log.Printf("Invalid messages: %d", kc.invalidCounter)
}

func (kc *KafkaConsumer) Consume() {
	kc.createConsumer()
	defer kc.shutdown()

	// Handle graceful shutdown
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)

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