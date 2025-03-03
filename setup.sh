#!/bin/bash

set -e 
PRODUCER_REPLICAS=${1:-1}
CONSUMER_REPLICAS=${2:-1}

echo "🧹 Starting cleanup..."

# Delete Kubernetes resources
echo "🗑️ Deleting Kubernetes resources..."
pushd deployments
kubectl delete -f producer-consumer.yaml --ignore-not-found=true
kubectl delete -f kafka-deployment.yaml --ignore-not-found=true
kubectl delete -f kafka-config.yaml --ignore-not-found=true
popd

# Remove Docker images
echo "🗑️ Removing Docker images..."
docker rmi -f producer-consumer:latest 2>/dev/null || true

echo "🏗️ Building and loading images into kind cluster..."
docker build -t producer-consumer:latest -f Dockerfile .
kind load docker-image producer-consumer:latest

pushd deployments
echo "⚙️ Applying config map..."
kubectl apply -f kafka-config.yaml

echo "✉️ Deploying Kafka..."
export KAFKA_BROKER_NAME=$(kubectl get configmap kafka-config -o jsonpath='{.data.KAFKA_BROKER_NAME}')
export KAFKA_BROKER_PORT=$(kubectl get configmap kafka-config -o jsonpath='{.data.KAFKA_BROKER_PORT}')
export KAFKA_TOPIC=$(kubectl get configmap kafka-config -o jsonpath='{.data.KAFKA_TOPIC}')
export KAFKA_DLQ_TOPIC=$(kubectl get configmap kafka-config -o jsonpath='{.data.KAFKA_DLQ_TOPIC}')

echo "✉️ Creating Kafka namespace"
kubectl create namespace kafka --dry-run=client -o yaml | kubectl apply -f -

echo "✉️ Applying Strimzi install files and some CRDs"
kubectl create -f 'https://strimzi.io/install/latest?namespace=kafka' -n kafka --dry-run=client -o yaml | kubectl apply -f -

echo "✉️ Applying Kafka cluster defined in .yaml file"
envsubst < kafka-deployment.yaml | kubectl apply -f -

echo "✉️ Waiting for Kafka cluster to be ready..."
kubectl wait kafka/${KAFKA_BROKER_NAME} --for=condition=Ready --timeout=300s -n kafka 

echo "🚀 Applying consumer and producer..."
PRODUCER_REPLICAS=$PRODUCER_REPLICAS CONSUMER_REPLICAS=$CONSUMER_REPLICAS envsubst < producer-consumer.yaml | kubectl apply -f -
popd

echo "✅ All services deployed successfully!"
