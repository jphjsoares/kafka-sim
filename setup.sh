#!/bin/bash
set -e

export $(grep -v '^#' .env | xargs)

PRODUCER_REPLICAS=${1:-1}
CONSUMER_REPLICAS=${2:-1}

echo "Cleaning up existing resources..."
kubectl delete deployment kafka-broker kafka-producer kafka-consumer --ignore-not-found
kubectl delete service kafka-broker --ignore-not-found

echo "Building and loading images into kind cluster..."
docker build -t producer-consumer:latest -f Dockerfile .
kind load docker-image producer-consumer:latest

echo "Deploying Kafka..."
envsubst < kafka-deployment.yaml | kubectl apply -f -

echo "Waiting for Kafka to be ready..."
kubectl wait --for=condition=available --timeout=60s deployment/kafka-broker

KAFKA_POD=$(kubectl get pods -l app=$KAFKA_BROKER_SERVICE_NAME -o jsonpath='{.items[0].metadata.name}')
create_topic() {
  TOPIC_NAME=$1
  echo "Creating Kafka topic: $TOPIC_NAME..."
  kubectl exec -it "$KAFKA_POD" -- /opt/kafka/bin/kafka-topics.sh --create \
    --bootstrap-server "localhost:$KAFKA_BROKER_PORT" \
    --replication-factor 1 \
    --partitions 10 \
    --topic "$TOPIC_NAME"
}

create_topic "$KAFKA_TOPIC"
create_topic "$KAFKA_DLQ_TOPIC"

echo "Deploying producer with $PRODUCER_REPLICAS replicas and consumer with $CONSUMER_REPLICAS replicas..."
PRODUCER_REPLICAS=$PRODUCER_REPLICAS CONSUMER_REPLICAS=$CONSUMER_REPLICAS envsubst < producer-consumer.yaml | kubectl apply -f -

echo "All services deployed successfully!"
