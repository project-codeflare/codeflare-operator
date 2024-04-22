#!/bin/bash

# Copyright 2024 IBM, Red Hat
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

# Installs a Kueue release on a cluster

set -euo pipefail
: "${KUEUE_VERSION}"

echo "Deploying Kueue version $KUEUE_VERSION"
kubectl apply --server-side -f https://github.com/kubernetes-sigs/kueue/releases/download/${KUEUE_VERSION}/manifests.yaml

# Sleep until the kueue manager is running
echo "Waiting for pods in the kueue-system namespace to become ready"
while [[ $(kubectl get pods -n kueue-system -o 'jsonpath={..status.conditions[?(@.type=="Ready")].status}' | tr ' ' '\n' | sort -u) != "True" ]]
do
    echo -n "." && sleep 1;
done
echo ""
