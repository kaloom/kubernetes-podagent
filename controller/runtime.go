/*
Copyright 2020 Kaloom Inc.
Copyright 2014 The Kubernetes Authors.

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

package controller

// Runtime interface
type Runtime interface {
	// GetNetNS returns the network namespace of the given containerID. The ID
	// supplied is typically the ID of a pod sandbox. This getter doesn't try
	// to map non-sandbox IDs to their respective sandboxes.
	GetNetNS(podSandboxID string) (string, error)

	// GetSandboxID returns kubernete's docker "pause" container ID
	GetSandboxID(containerID string) (string, error)
}
