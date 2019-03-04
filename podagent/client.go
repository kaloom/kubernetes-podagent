/*
 Copyright (c) Kaloom, 2017-2019

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

package main

import (
	"fmt"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func createClient(kubeconfig string) (*kubernetes.Clientset, error) {
	var err error

	cfg := &rest.Config{}
	if kubeconfig != "" {
		// get a config from the provided kubeconfig file and use the current context
		cfg, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("can't build kube config off %v", err)
		}
	} else {
		// get a config from within the pod for in-cluster authentication
		cfg, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("couldn't initialize InClusterConfig %v", err)
		}
	}

	// creates the clientset
	return kubernetes.NewForConfig(cfg)
}
