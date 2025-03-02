# Setup

```bash
kind create cluster

# Setup kubernetes cluster
chmod +x setup.sh
./setup.sh <producers_replicas> <consumers_replicas>

# To delete cluster
kind delete cluster
```

# Executing

Now, consumers and producers are exchanging messages. Use the `monitor.py` to get the stats:

```bash
python3 -m venv env/
source env/bin/activate
python3 -m pip install kubernetes
python3 monitor.py
```
