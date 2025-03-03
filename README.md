# Setup

```bash
# Create a cluster
kind create cluster

# Note: if it's the first time running, you must apply the config map first, in order for the setup.sh to not fail
pushd deployments
kubectl apply -f kafka-config.yaml
popd

# Setup kubernetes cluster
chmod +x setup.sh
./setup.sh <producers_replicas> <consumers_replicas>

# To delete cluster
kind delete cluster
```
