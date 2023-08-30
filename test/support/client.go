/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package support

import (
	mcadclient "github.com/project-codeflare/multi-cluster-app-dispatcher/pkg/client/clientset/versioned"
	rayclient "github.com/ray-project/kuberay/ray-operator/pkg/client/clientset/versioned"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/tools/clientcmd"

	imagev1 "github.com/openshift/client-go/image/clientset/versioned"
	routev1 "github.com/openshift/client-go/route/clientset/versioned"

	codeflareclient "github.com/project-codeflare/codeflare-operator/client/clientset/versioned"
	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
)

type Client interface {
	Core() kubernetes.Interface
	Route() routev1.Interface
	Image() imagev1.Interface
	CodeFlare() codeflareclient.Interface
	MCAD() mcadclient.Interface
	Ray() rayclient.Interface
	Dynamic() dynamic.Interface
}

type testClient struct {
	core      kubernetes.Interface
	route     routev1.Interface
	image     imagev1.Interface
	codeflare codeflareclient.Interface
	mcad      mcadclient.Interface
	ray       rayclient.Interface
	dynamic   dynamic.Interface
}

var _ Client = (*testClient)(nil)

func (t *testClient) Core() kubernetes.Interface {
	return t.core
}

func (t *testClient) Route() routev1.Interface {
	return t.route
}

func (t *testClient) Image() imagev1.Interface {
	return t.image
}

func (t *testClient) CodeFlare() codeflareclient.Interface {
	return t.codeflare
}

func (t *testClient) MCAD() mcadclient.Interface {
	return t.mcad
}

func (t *testClient) Ray() rayclient.Interface {
	return t.ray
}

func (t *testClient) Dynamic() dynamic.Interface {
	return t.dynamic
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

	routeClient, err := routev1.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	imageClient, err := imagev1.NewForConfig(cfg)
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

	rayClient, err := rayclient.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	dynamicClient, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	return &testClient{
		core:      kubeClient,
		route:     routeClient,
		image:     imageClient,
		codeflare: codeFlareClient,
		mcad:      mcadClient,
		ray:       rayClient,
		dynamic:   dynamicClient,
	}, nil
}
