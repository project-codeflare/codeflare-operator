package main

import (
	"fmt"

	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

func createClientSet() *kubernetes.Clientset{
	k8sConfig = config.GetConfigOrDie()
	clientSet, err := kubernetes.NewForConfig(k8sConfig)

	// Creating client Set
	if err != nil {
		fmt.Errorf("Error creating client set: ", err.Error())
	}
	return clientSet
}
