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
	"context"
	"flag"
	"fmt"

	"github.com/kaloom/kubernetes-podagent/controller"
)

var (
	branch = "unknown"
	commit = "unknown"
	date   = "unknown"
)

func showBuildDetails() {
	fmt.Printf("podagent build details, branch/tag: %s, commit: %s, date: %s\n", branch, commit, date)
}

func main() {
	kubeconfig := flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	nodeName := flag.String("node", "", "kubernetes node name")
	dockerEndpoint := flag.String("docker-endpoint", "unix:///var/run/docker.sock", "docker endpoint")
	cniBinPath := flag.String("cni-bin-path", "/opt/cni/bin", "cni plugin binary path")
	cniConfPath := flag.String("cni-conf-path", "/etc/cni/net.d", "cni plugin network configuration path")
	cniVendorName := flag.String("cni-vendor-name", "", "cni vendor name (default \"\", i.e. use the cni-plugin type found off the first lexical config in /etc/cni/net.d)")
	showVersion := flag.Bool("version", false, "display build details and exist")
	flag.Parse()

	if *showVersion {
		showBuildDetails()
		return
	}
	if *nodeName == "" {
		fmt.Printf("The node name as registered by kubelet over kube-apiserver must be provided via the -node command-line argument\n")
		return
	}
	kubeClient, err := createClient(*kubeconfig)
	if err != nil {
		fmt.Printf("Failed to create kubernetes client: %v\n", err)
		return
	}
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()

	controller, err := controller.NewController(kubeClient, *dockerEndpoint, *cniBinPath, *cniConfPath, *cniVendorName)
	if err != nil {
		fmt.Printf("Failed to create a controller: %v\n", err)
		return
	}

	showBuildDetails()
	controller.Run(ctx, *nodeName)
}
