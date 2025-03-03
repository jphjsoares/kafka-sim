package kafka

import (
	"context"
	"encoding/json"
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
	broker       []string
	topic        string
	dlqTopic     string
	reader       *kafka.Reader
	dlqWriter    *kafka.Writer
	running      bool
	validCounter   int
	invalidCounter int
}


// TODO:
// This is still not working (not consuming), so I will need to fix it later.
func NewKafkaConsumer(brokers []string, topic string, dlqTopic string) (*KafkaConsumer, error) {
	return &KafkaConsumer{
		brokers, 
		topic, 
		dlqTopic,
		kafka.NewReader(kafka.ReaderConfig{
			Brokers:   brokers,
			GroupID:   "myconsumergroup",
			Topic:     topic,
			MinBytes:  10e3,
			MaxBytes:  10e6,
		}),
		&kafka.Writer{
			Addr:     kafka.TCP(brokers...),
			Topic:    dlqTopic,
			Balancer: &kafka.LeastBytes{},
		}, 
		true, 
		0, 
		0,
	}, nil
}

func (kc *KafkaConsumer) Consume() {
	log.Println("Consumer started and waiting for messages...")
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		log.Println("Shutdown signal received")
		kc.running = false
		kc.reader.Close()
		kc.dlqWriter.Close()
	}()

	for kc.running {
		msg, err := kc.reader.ReadMessage(context.Background())
		if err != nil {
			log.Printf("Consumer error: %v", err)
			break
		}
		kc.processMessage(msg)
	}
}

func (kc *KafkaConsumer) processMessage(msg kafka.Message) {
	message := &protobuf.Message{}
	if err := proto.Unmarshal(msg.Value, message); err != nil {
		kc.sendToDLQ(msg, err)
		return
	}

	if message.Id == "" || message.Content == "" || message.Timestamp == "" {
		err := log.Output(1, "Invalid protobuf message structure")
		kc.sendToDLQ(msg, err)
		return
	}

	log.Printf("Consumed valid message: %v", message)
	kc.validCounter++
	log.Printf("Valid messages: %d", kc.validCounter)
	log.Printf("Invalid messages: %d", kc.invalidCounter)
}

func (kc *KafkaConsumer) sendToDLQ(msg kafka.Message, err error) {
	failedMessage := map[string]interface{}{
		"original_message": string(msg.Value),
		"error":            err.Error(),
		"timestamp":        time.Now().Format(time.RFC3339),
	}
	encoded, _ := json.Marshal(failedMessage)
	kc.dlqWriter.WriteMessages(context.Background(), kafka.Message{
		Value: encoded,
	})
	kc.invalidCounter++
	log.Printf("Sent to DLQ: %s", encoded)
}
