/*
Copyright 2017-2019 Kaloom Inc.
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

package dockerruntime

import (
	"fmt"

	"github.com/blang/semver"
	dockertypes "github.com/docker/docker/api/types"
	"github.com/golang/glog"

	"k8s.io/kubernetes/pkg/kubelet/util/cache"
)

const (
	dockerNetNSFmt = "/proc/%v/ns/net"
)

// DockerRuntime docker runtime object
type DockerRuntime struct {
	client libdocker.Interface
	// caches the version of the runtime.
	// To be compatible with multiple docker versions, we need to perform
	// version checking for some operations. Use this cache to avoid querying
	// the docker daemon every time we need to do such checks.
	versionCache *cache.ObjectCache
}

func getNetworkNamespace(c *dockertypes.ContainerJSON) (string, error) {
	if c.State.Pid == 0 {
		// Docker reports pid 0 for an exited container.
		return "", fmt.Errorf("Cannot find network namespace for the terminated container %q", c.ID)
	}
	return fmt.Sprintf(dockerNetNSFmt, c.State.Pid), nil
}

// GetNetNS returns the network namespace of the given containerID. The ID
// supplied is typically the ID of a pod sandbox. This getter doesn't try
// to map non-sandbox IDs to their respective sandboxes.
func (dr *DockerRuntime) GetNetNS(podSandboxID string) (string, error) {
	c, err := dr.client.InspectContainer(podSandboxID)
	if err != nil {
		return "", err
	}

	ns, err := getNetworkNamespace(c)
	glog.V(5).Infof("GetNetNS:%s %v", ns, err)
	return ns, err
}

// GetSandboxID returns kubernete's docker "pause" container ID
func (dr *DockerRuntime) GetSandboxID(containerID string) (string, error) {
	const kubernetesSandboxID = "io.kubernetes.sandbox.id"
	c, err := dr.client.InspectContainer(containerID)
	if err != nil {
		return "", err
	}
	if len(c.Config.Labels) > 0 {
		if val, ok := c.Config.Labels[kubernetesSandboxID]; ok {
			return val, nil
		}
	}
	glog.V(5).Infof("GetSandboxID:SandboxId %s", kubernetesSandboxID)
	return "", fmt.Errorf("Cannot find label %s in container %q", kubernetesSandboxID, c.ID)
}

// dockerVersion gets the version information from docker.
func (dr *DockerRuntime) getDockerVersion() (*dockertypes.Version, error) {
	v, err := dr.client.Version()
	if err != nil {
		return nil, fmt.Errorf("failed to get docker version: %v", err)
	}
	// Docker API version (e.g., 1.23) is not semver compatible. Add a ".0"
	// suffix to remedy this.
	v.APIVersion = fmt.Sprintf("%s.0", v.APIVersion)
	return v, nil
}

func (dr *DockerRuntime) getDockerVersionFromCache() (*dockertypes.Version, error) {
	// We only store on key in the cache.
	const dummyKey = "version"
	value, err := dr.versionCache.Get(dummyKey)
	dv := value.(*dockertypes.Version)
	if err != nil {
		return nil, err
	}
	return dv, nil
}

// getDockerAPIVersion gets the semver-compatible docker api version.
func (dr *DockerRuntime) getDockerAPIVersion() (*semver.Version, error) {
	var dv *dockertypes.Version
	var err error
	if dr.versionCache != nil {
		dv, err = dr.getDockerVersionFromCache()
	} else {
		dv, err = dr.getDockerVersion()
	}
	if err != nil {
		return nil, err
	}

	apiVersion, err := semver.Parse(dv.APIVersion)
	if err != nil {
		return nil, err
	}
	return &apiVersion, nil
}

// checkVersionCompatibility verifies whether docker is in a compatible version.
func (dr *DockerRuntime) checkVersionCompatibility() error {
	apiVersion, err := dr.getDockerAPIVersion()
	if err != nil {
		return err
	}

	minAPIVersion, err := semver.Parse(libdocker.MinimumDockerAPIVersion)
	if err != nil {
		return err
	}

	// Verify the docker version.
	result := apiVersion.Compare(minAPIVersion)
	if result < 0 {
		return fmt.Errorf("docker API version is older than %s", libdocker.MinimumDockerAPIVersion)
	}

	return nil
}

// NewDockerRuntime instantiate a docker runtime object
func NewDockerRuntime(client libdocker.Interface) (*DockerRuntime, error) {
	c := libdocker.NewInstrumentedInterface(client)

	dr := &DockerRuntime{
		client: c,
	}

	if dockerInfo, err := dr.client.Info(); err != nil {
		glog.Errorf("Failed to execute Info() call to the Docker client: %v", err)
	} else {
		glog.Infof("Docker client info: server version %s, is an experimental build? %v",
			dockerInfo.ServerVersion, dockerInfo.ExperimentalBuild)
	}

	// check docker version compatibility.
	if err := dr.checkVersionCompatibility(); err != nil {
		return nil, err
	}

	return dr, nil
}
