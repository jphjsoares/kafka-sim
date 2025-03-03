package kafka

import (
	"fmt"
	"net/http"

	"github.com/segmentio/kafka-go"
)

var (
	kafkaBrokers []string
	kafkaTopic   string
)

func CheckKafkaCon() error {
	// Create a Kafka connection to check if Kafka is reachable
	conn, err := kafka.Dial("tcp", kafkaBrokers[0])
	if err != nil {
		return fmt.Errorf("failed to connect to Kafka: %v", err)
	}
	defer conn.Close()

	// Check if the topic exists
	partitions, err := conn.ReadPartitions(kafkaTopic)
	if err != nil {
		return fmt.Errorf("failed to read partitions for topic %s: %v", kafkaTopic, err)
	}
	if len(partitions) == 0 {
		return fmt.Errorf("no partitions found for topic %s", kafkaTopic)
	}

	return nil
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	if err := CheckKafkaCon(); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(w, "unhealthy: %v", err)
		return
	}
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "healthy")
}

func StartHealthServer(brokers []string, topic string) {
	kafkaBrokers = brokers
	kafkaTopic = topic
	http.HandleFunc("/health", healthHandler)
	http.ListenAndServe(":8080", nil)
}

