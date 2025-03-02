from confluent_kafka import Consumer, Producer, KafkaError
import os
import logging
import json 
from jsonschema import validate
from datetime import datetime
import time
import signal

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger("consumer")

class KafkaConsumer:
    def __init__(self, kafka_broker_host, kafka_broker_port, dlq_topic, group_id="myconsumergroup"):
        self.kafka_broker_server = f"{kafka_broker_host}:{kafka_broker_port}"
        self.dlq_topic = dlq_topic
        self.group_id = group_id
        self.consumer = None
        self.dlq_producer = None
        self.schema = None
        self.valid_counter = 0 
        self.invalid_counter = 0 
        self.running = True
        signal.signal(signal.SIGTERM, self.shutdown) # from kubernetes
        self.load_schema("message_schema.json")

    def load_schema(self, schema_file_path):
        # get message schema, to enforce structured messages
        with open(schema_file_path) as f:
            self.schema = json.load(f)

    def shutdown(self):
        # ensure graceful shutdowns
        logger.info("Shutdown received. Closing consumer...")
        self.running = False
        self.consumer.close()

    def create_consumer(self, topic):
        conf = {
            "bootstrap.servers": self.kafka_broker_server,
            "group.id": self.group_id, # consumers in the same group share messages from partitions
            "auto.offset.reset": "earliest", # start reading earliest messages in the topic
            "enable.auto.commit": False # consumer commits manually, avoiding data loss or duplication
        }
        self.consumer = Consumer(conf)
        self.consumer.subscribe([topic]) # start listening for messages in a topic

        # Create DLQ producer
        self.dlq_producer = Producer({"bootstrap.servers": self.kafka_broker_server})

    def send_to_dlq(self, msg, error):
        failed_message = {
            "original_message": msg.value().decode("utf-8"),
            "error": str(error),
            "timestamp": datetime.now().isoformat()
        }
        self.dlq_producer.produce(self.dlq_topic, json.dumps(failed_message).encode("utf-8"))
        self.dlq_producer.flush()
        self.invalid_counter += 1

    def process_message(self, msg):
        try:
            message = json.loads(msg.value().decode("utf-8"))
            validate(instance=message, schema=self.schema)
            logger.info(f"Consumed: {msg.value().decode('utf-8')}")
            time.sleep(0.05) # simulate any processing that would happen here
            self.consumer.commit(msg)  # ensure message commit reliability
            self.valid_counter += 1
            logger.info(f"Consumed messages:{self.valid_counter}")
            logger.info(f"Invalid messages:{self.invalid_counter}")
        except Exception as e:
            self.send_to_dlq(msg, e)

    def consume(self):
        logger.info(f"Consumer started and waiting for messages...")
        try:
            while self.running:
                msg = self.consumer.poll(5)
                if msg is None:
                    continue
                if msg.error():
                    # reached the end of partition
                    if msg.error().code() == KafkaError._PARTITION_EOF:
                        continue
                    logger.error(f"Consumer error: {msg.error()}")
                    break
                self.process_message(msg)
        finally:
            self.consumer.close()

if __name__ == "__main__":
    kafka_broker_host = os.getenv("KAFKA_BROKER_HOST")
    kafka_broker_port = os.getenv("KAFKA_BROKER_PORT")
    topic = os.getenv("KAFKA_TOPIC")
    dlq_topic = os.getenv("KAFKA_DLQ_TOPIC")
    
    consumer = KafkaConsumer(kafka_broker_host, kafka_broker_port, dlq_topic)
    consumer.create_consumer(topic)
    consumer.consume()
