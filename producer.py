from confluent_kafka import Producer
from confluent_kafka.admin import AdminClient
import os
import json
from datetime import datetime
import uuid
import time
import logging
import random

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger("producer")

class KafkaProducer:
    def __init__(self, kafka_broker_host, kafka_broker_port, topic):
        self.kafka_broker_server = f"{kafka_broker_host}:{kafka_broker_port}"
        self.topic = topic
        self.producer = None
        self.admin_client = None
        self.valid_counter = 0
        self.invalid_counter = 0
        self.schema = None
        self.load_schema("message_schema.json")

    def load_schema(self, schema_file_path):
        with open(schema_file_path) as f:
            self.schema = json.load(f)

    def create_producer(self):
        conf = {
            "bootstrap.servers": f"{self.kafka_broker_server}",
        }
        self.producer = Producer(conf)
        self.admin_client = AdminClient(conf)

    def delivery_report(self, err, msg):
        if err is not None:
            logger.error(f"Message delivery failed: {err}")
        else:
            logger.info(f"Produced messages:{self.valid_counter}")
            logger.info(f"Invalid messages:{self.invalid_counter}") 
            logger.info(f"Produced: {msg.value().decode('utf-8')}")

    def create_message(self):
        return {
            "id": str(uuid.uuid4()),
            "content": f"Hello corti platform team!",
            "timestamp": datetime.now().isoformat()
        }

    def generate_messages(self):
        logger.info("Start sending messages bursts...")
        while True:
            # 50 message burst
            for _ in range(50):
                # randomly generate an invalid message
                if random.random() < 0.05:
                    self.invalid_counter += 1
                    message = {"invalid_field": "Should be sent to DLQ!"}
                else:

                    self.valid_counter += 1
                    message = self.create_message()
                serialized_message = json.dumps(message).encode("utf-8")
                try:
                    self.producer.produce(self.topic, serialized_message, callback=self.delivery_report)
                except Exception as kafka_e:
                    logger.error(f"Kafka error: {kafka_e}")
            self.producer.flush()
            time.sleep(1)

if __name__ == "__main__":
    kafka_broker_host = os.getenv("KAFKA_BROKER_HOST")
    kafka_broker_port = os.getenv("KAFKA_BROKER_PORT")
    topic = os.getenv("KAFKA_TOPIC")
    
    producer = KafkaProducer(kafka_broker_host, kafka_broker_port, topic)
    producer.create_producer()
    producer.generate_messages()
