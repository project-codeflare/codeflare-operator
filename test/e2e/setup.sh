#!/bin/bash

# Copyright 2022 IBM, Red Hat
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -euo pipefail
: "${KUBERAY_VERSION}"

echo Deploying KubeRay "${KUBERAY_VERSION}"
kubectl apply --server-side -k "github.com/ray-project/kuberay/ray-operator/config/default?ref=${KUBERAY_VERSION}&timeout=180s"

cat <<EOF | kubectl apply -f -
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: e2e-controller-rayclusters
rules:
  - apiGroups:
      - ray.io
    resources:
      - rayclusters
      - rayclusters/finalizers
      - rayclusters/status
    verbs:
      - get
      - list
      - watch
      - create
      - update
      - patch
      - delete
EOF

cat <<EOF | kubectl apply -f -
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: e2e-controller-rayclusters
subjects:
  - kind: ServiceAccount
    name: codeflare-operator-controller-manager
    namespace: openshift-operators
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: e2e-controller-rayclusters
EOF

echo "Deploying Kueue $KUEUE_VERSION"
kubectl apply --server-side -f https://github.com/kubernetes-sigs/kueue/releases/download/${KUEUE_VERSION}/manifests.yaml

# Sleep until the kueue manager is running
echo "Waiting for pods in the kueue-system namespace to become ready"
while [[ $(kubectl get pods -n kueue-system -o 'jsonpath={..status.conditions[?(@.type=="Ready")].status}' | tr ' ' '\n' | sort -u) != "True" ]]
do
    echo -n "." && sleep 1;
done
echo ""

echo Creating Kueue ResourceFlavor and ClusterQueue
cat <<EOF | kubectl apply -f -
apiVersion: kueue.x-k8s.io/v1beta1
kind: ResourceFlavor
metadata:
  name: "default-flavor"
EOF

cat <<EOF | kubectl apply -f -
apiVersion: kueue.x-k8s.io/v1beta1
kind: ClusterQueue
metadata:
  name: "e2e-cluster-queue"
spec:
  namespaceSelector: {} # match all.
  resourceGroups:
  - coveredResources: ["cpu","memory", "nvidia.com/gpu"]
    flavors:
    - name: "default-flavor"
      resources:
      - name: "cpu"
        nominalQuota: 4
      - name: "memory"
        nominalQuota: "20G"
      - name: "nvidia.com/gpu"
        nominalQuota: "1"
EOF
