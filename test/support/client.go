/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package support

import (
	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	codeflareclient "github.com/project-codeflare/codeflare-operator/client/clientset/versioned"
	mcadclient "github.com/project-codeflare/multi-cluster-app-dispatcher/pkg/client/clientset/controller-versioned"
)

type Client interface {
	Core() kubernetes.Interface
	CodeFlare() codeflareclient.Interface
	MCAD() mcadclient.Interface
}

type testClient struct {
	core      kubernetes.Interface
	codeflare codeflareclient.Interface
	mcad      mcadclient.Interface
}

var _ Client = (*testClient)(nil)

func (t *testClient) Core() kubernetes.Interface {
	return t.core
}

func (t *testClient) CodeFlare() codeflareclient.Interface {
	return t.codeflare
}

func (t *testClient) MCAD() mcadclient.Interface {
	return t.mcad
}

func newTestClient() (Client, error) {
	cfg, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	).ClientConfig()
	if err != nil {
		return nil, err
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	codeFlareClient, err := codeflareclient.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	mcadClient, err := mcadclient.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	return &testClient{
		core:      kubeClient,
		codeflare: codeFlareClient,
		mcad:      mcadClient,
	}, nil
}
