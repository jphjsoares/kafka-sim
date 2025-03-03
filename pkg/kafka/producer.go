package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/jphjsoares/kafka-sim/pkg/protobuf"
	"github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/proto"
)

type KafkaProducer struct {
	brokers      []string
	topic        string
	writer       *kafka.Writer
	validCounter int
	invalidCounter int
}

func NewKafkaProducer(brokers []string, topic string) *KafkaProducer {
	return &KafkaProducer{
		brokers: brokers,
		topic:   topic,
		writer: &kafka.Writer{
			Addr:     kafka.TCP(brokers...),
			Topic:    topic,
			Balancer: &kafka.LeastBytes{},
		},
	}
}

func (kp *KafkaProducer) createMessage() *protobuf.Message {
	return &protobuf.Message{
		// TODO : use uuid
		Id:        fmt.Sprintf("%d", rand.Intn(1000000)), 
		Content:   "Hello World!",
		Timestamp: time.Now().Format(time.RFC3339),
	}
}

func (kp *KafkaProducer) GenerateMessages() {
	log.Println("Start sending message bursts...")
	for {
		// Send 50 messages in a burst
		for range 50 {
			var message proto.Message
			if rand.Float32() < 0.05 { // 5% chance of invalid message
				kp.invalidCounter++
				invalidMessage := map[string]string{"invalid_field": "Should be sent to DLQ!"}
				messageBytes, _ := json.Marshal(invalidMessage)
				kp.writer.WriteMessages(context.Background(), kafka.Message{
					Value: messageBytes,
				})
				log.Printf("Produced invalid message: %s", invalidMessage)
			} else {
				kp.validCounter++
				message = kp.createMessage()
				messageBytes, _ := proto.Marshal(message)
				kp.writer.WriteMessages(context.Background(), kafka.Message{
					Value: messageBytes,
				})
				log.Printf("Produced valid message: %v", message)
			}
		}
		log.Printf("Produced messages: %d", kp.validCounter)
		log.Printf("Invalid messages: %d", kp.invalidCounter)
		time.Sleep(1 * time.Second)
	}
}

func (kp *KafkaProducer) Close() {
	kp.writer.Close()
}
