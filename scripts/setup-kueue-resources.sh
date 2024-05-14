#!/bin/bash

# Function to prompt the user for input with a default value
prompt_for_input() {
    local prompt_message=$1
    local default_value=$2
    local user_input

    read -p "$prompt_message [$default_value] ): " user_input
    echo "${user_input:-$default_value}"
}

echo "ClusterQueue Configuration"

# Using function to get user inputs
name=$(prompt_for_input "Enter your Cluster Queue name, (default : " "cluster-queue")
cpu=$(prompt_for_input "Enter your Cluster Queue CPU (The amount of CPUs available: (default: " "16")
memory=$(prompt_for_input "Enter your Cluster Queue Memory (The amount of memory available) (default: " "60Gi")
gpu=$(prompt_for_input "Enter your Cluster Queue GPU count (The amount of GPUs available) (default: " "0")
pods=$(prompt_for_input "Enter your Cluster Queue Pods limit (The maximum amount of pods that can be created) (default:" "10")
flavour=$(prompt_for_input "Enter your Cluster Queue Resource Flavour (default: " "default-flavor")

# Applying ClusterQueue configuration
kubectl apply --server-side -f - <<EOF
apiVersion: kueue.x-k8s.io/v1beta1
kind: ClusterQueue
metadata:
    name: $name
spec:
  namespaceSelector: {} # match all.
  resourceGroups:
  - coveredResources: ["cpu", "memory", "pods", "nvidia.com/gpu"]
    flavors:
    - name: $flavour
      resources:
      - name: "cpu"
        nominalQuota: $cpu
      - name: "memory"
        nominalQuota: $memory
      - name: "pods"
        nominalQuota: $pods
      - name: "nvidia.com/gpu"
        nominalQuota: $gpu
EOF
echo "Cluster Queue $name applied!"

# Applying ResourceFlavor configuration
echo "Applying Resource Flavour"
kubectl apply --server-side -f - <<EOF
apiVersion: kueue.x-k8s.io/v1beta1
kind: ResourceFlavor
metadata:
    name: $flavour
EOF
echo "Resource Flavour $flavour applied!"

# Applying LocalQueue configuration
echo "LocalQueue Configuration"
lq_name=$(prompt_for_input "Enter your Local Queue name (default: " "local-queue-default")
namespace=$(prompt_for_input "Enter your Local Queue namespace (default: " "default")
default=$(prompt_for_input "Is this the default LQ (default: " "false")

kubectl apply --server-side -f - <<EOF
apiVersion: kueue.x-k8s.io/v1beta1
kind: LocalQueue
metadata:
    namespace: $namespace
    name: $lq_name
    annotations:
      "kueue.x-k8s.io/default-queue": "$default"
spec:
  clusterQueue: $name
EOF
echo "Local Queue $lq_name applied!"


echo "Configuration applied"
