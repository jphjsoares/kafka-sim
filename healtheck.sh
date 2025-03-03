#!/bin/bash

set -e

if [ "$1" == "consumer" ]; then
  APP_LABEL="kafka-consumer"
elif [ "$1" == "producer" ]; then
  APP_LABEL="kafka-producer"
else
  echo "Usage: $0 [consumer|producer]"
  exit 1
fi

PODS=$(kubectl get pod -l app=$APP_LABEL -o jsonpath="{.items[*].metadata.name}")

if [ -z "$PODS" ]; then
  echo "No $1 pods found"
  exit 1
fi

for POD_NAME in $PODS; do
  echo "Checking health for pod: $POD_NAME"

  kubectl port-forward $POD_NAME 8080:8080 &
  PORT_FORWARD_PID=$!

  # Give the port-forward some time to establish
  sleep 2

  HEALTH_RESPONSE=$(curl -s http://localhost:8080/health)

  if [[ "$HEALTH_RESPONSE" == *"healthy"* ]]; then
    echo "✅ $POD_NAME: healthy"
  else
    echo "❌ $POD_NAME: Health check failed"
    echo "$HEALTH_RESPONSE"
  fi

  kill $PORT_FORWARD_PID
  sleep 2
done