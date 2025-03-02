from kubernetes import client, config
import re

config.load_kube_config()
v1 = client.CoreV1Api()

def fetch_pod_logs(namespace, pod_name, container_name):
    try:
        logs = v1.read_namespaced_pod_log(
            name=pod_name,
            namespace=namespace,
            container=container_name,
            follow=False,
            tail_lines=10
        )
        return logs 
    except Exception as e:
        print(f"Failed to fetch logs from pod {pod_name}: {e}")
        return None

def get_pods_by_label(namespace, label_selector):
    try:
        pods = v1.list_namespaced_pod(
            namespace=namespace,
            label_selector=label_selector
        )
        return pods.items
    except Exception as e:
        print(f"Failed to list pods: {e}")
        return []

def extract_latest_count(logs, prefix):
    # Example: "Produced messages:42" or "Consumed messages:99"
    pattern = re.compile(rf"{prefix} messages:(\d+)")
    matches = pattern.findall(logs)
    if matches:
        return int(matches[-1])
    return 0

def extract_dlq_count(logs):
    pattern = re.compile(r"Invalid messages:(\d+)")
    matches = pattern.findall(logs)
    if matches:
        return int(matches[-1])
    return 0

def main():
    namespace = "default"
    consumer_label_selector = "app=kafka-consumer"
    producer_label_selector = "app=kafka-producer"

    consumer_pods = get_pods_by_label(namespace, consumer_label_selector)
    total_consumed = 0
    total_dlq_messages = 0
    total_invalid_messages = 0
    for pod in consumer_pods:
        logs = fetch_pod_logs(namespace, pod.metadata.name, "kafka-consumer") 
        if logs:
            consumed_count = extract_latest_count(logs, "Consumed")
            dlq_count = extract_dlq_count(logs)
            print(f"{pod.metadata.name} consumed {consumed_count}")
            print(f"{pod.metadata.name} DLQ messages: {dlq_count}")
            total_consumed += consumed_count
            total_dlq_messages += dlq_count

    producer_pods = get_pods_by_label(namespace, producer_label_selector)
    total_produced = 0
    for pod in producer_pods:
        logs = fetch_pod_logs(namespace, pod.metadata.name, "kafka-producer")
        if logs:
            produced_count = extract_latest_count(logs, "Produced")
            invalid_messages = extract_dlq_count(logs)
            print(f"{pod.metadata.name} produced {produced_count}")
            print(f"{pod.metadata.name} invalid messages produced: {invalid_messages}") 
            total_produced += produced_count
            total_invalid_messages += invalid_messages


    print(f"\nTotal Produced: {total_produced}")
    print(f"Total invalid messaged produced: {total_invalid_messages}")

    print(f"\nTotal Consumed: {total_consumed}")
    print(f"Total sent to dlq: {total_dlq_messages}")
    consumption_rate = (total_consumed / total_produced * 100) if total_produced > 0 else 0
    dlq_invalid_rate = (total_dlq_messages / total_invalid_messages * 100) if total_invalid_messages > 0 else 0
    print(f"Consumption Success Rate: {consumption_rate:.2f}%")
    print(f"Sent to DLQ Rate: {dlq_invalid_rate:.2f}%")

if __name__ == "__main__":
    main()
