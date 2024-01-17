// /*
// Copyright 2020 Kaloom Inc.
// Copyright 2014 The Kubernetes Authors.

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//     http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

package crioruntime

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	pb "k8s.io/cri-api/pkg/apis/runtime/v1"
	"k8s.io/kubernetes/pkg/kubelet/util"
)

const (
	crioNetNSFmt = "/var/run/netns/%s"
)

// CrioRuntime runtime object
type CrioRuntime struct {
	client pb.RuntimeServiceClient
}

type PodStatusResponseInfo struct {
	SandboxId   string
	RunTimeSpec RuneTimeSpecInfo
}

type RuneTimeSpecInfo struct {
	Linux NamespacesInfo
}

type NamespacesInfo struct {
	NameSpaces []NameSpaceInfo
}

type NameSpaceInfo struct {
	Type string
	Path string
}

// GetNetNS returns the network namespace of the given containerID. The ID
// supplied is typically the ID of a pod sandbox. This getter doesn't try
// to map non-sandbox IDs to their respective sandboxes.
func (cr *CrioRuntime) GetNetNS(podSandboxID string) (string, error) {

	glog.V(4).Infof("GetNetNS:podSandboxID:%s", podSandboxID)
	if podSandboxID == "" {
		return "", fmt.Errorf("ID cannot be empty")
	}

	request := &pb.PodSandboxStatusRequest{
		PodSandboxId: podSandboxID,
		Verbose:      true, // TODO see with non verbose if all info is there
	}
	glog.V(5).Infof("PodSandboxStatusRequest: %v", request)
	r, err := cr.client.PodSandboxStatus(context.Background(), request)
	glog.V(5).Infof("PodSandboxStatusResponse: %v", r)
	if err != nil {
		return "", err
	}

	mapInfo := r.GetInfo()
	glog.V(5).Infof("GetNetNS:GetInfo():%s", mapInfo)
	var podStatusResponseInfo PodStatusResponseInfo
	info := mapInfo["info"]
	glog.V(5).Infof("GetNetNS:info:%s", info)
	err = json.Unmarshal([]byte(info), &podStatusResponseInfo)
	if err != nil {
		glog.Errorf("GetNetNS:error decoding response: %v", err)
		if e, ok := err.(*json.SyntaxError); ok {
			glog.Errorf("GetNetNS:syntax error at byte offset %d", e.Offset)
		}
		return "", err
	}

	namespaces := podStatusResponseInfo.RunTimeSpec.Linux.NameSpaces
	glog.V(5).Infof("GetNetNS:RunTimeSpec.Linux.NameSpaces: %v", namespaces)
	for _, namespace := range namespaces {
		if namespace.Type == "network" {
			ss := strings.Split(namespace.Path, "/")
			netNS := ss[len(ss)-1]
			glog.V(5).Infof("GetNetNS:NetNS:%s", netNS)
			return fmt.Sprintf(crioNetNSFmt, netNS), nil
		}
	}
	return "", nil
}

// GetSandboxID returns kubernete's crio sandbox container ID
func (cr *CrioRuntime) GetSandboxID(containerID string) (string, error) {
	glog.V(5).Infof("GetSandboxID:containerID:%s", containerID)
	if containerID == "" {
		return "", fmt.Errorf("ID cannot be empty")
	}

	filter := &pb.ContainerFilter{
		Id: containerID,
	}

	request := &pb.ListContainersRequest{
		Filter: filter,
	}

	glog.V(5).Infof("ListContainerRequest: %v", request)
	r, err := cr.client.ListContainers(context.Background(), request)
	glog.V(5).Infof("ListContainerResponse: %v", r)
	if err != nil {
		return "", err
	}

	containerslist := r.GetContainers()
	if len(containerslist) == 0 {
		return "", fmt.Errorf("Didn't find any container with containerID:%s", containerID)
	} else if len(containerslist) != 1 {
		return "", fmt.Errorf("Found more then one container with containerID:%s", containerID)
	}

	sandboxID := containerslist[0].PodSandboxId
	glog.V(5).Infof("ContainerStatusResponse:SandboxId %s", sandboxID)
	return sandboxID, nil
}

func getConnection(endPoints []string, timeOut time.Duration) (*grpc.ClientConn, error) {
	if endPoints == nil || len(endPoints) == 0 {
		return nil, fmt.Errorf("endpoint is not set")
	}
	endPointsLen := len(endPoints)
	var conn *grpc.ClientConn
	for indx, endPoint := range endPoints {
		glog.Infof("connect using endpoint '%s' with '%s' timeout", endPoint, timeOut)
		addr, dialer, err := util.GetAddressAndDialer(endPoint)
		if err != nil {
			if indx == endPointsLen-1 {
				return nil, err
			}
			glog.Error(err)
			continue
		}
		conn, err = grpc.Dial(addr, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(timeOut), grpc.WithContextDialer(dialer))
		if err != nil {
			errMsg := errors.Wrapf(err, "connect endpoint '%s', make sure you are running as root and the endpoint has been started", endPoint)
			if indx == endPointsLen-1 {
				return nil, errMsg
			}
			glog.Error(errMsg)
		} else {
			glog.Infof("connected successfully using endpoint: %s", endPoint)
			break
		}
	}
	return conn, nil
}

// NewCrioRuntime instantiate a crio runtime object
func NewCrioRuntime(endpoint string, timeOut time.Duration) (*CrioRuntime, error) {

	if endpoint == "" {
		return nil, fmt.Errorf("--runtime-endpoint is not set")
	}
	clientConnection, err := getConnection([]string{endpoint}, timeOut)
	if err != nil {
		return nil, errors.Wrap(err, "connect")
	}
	runtimeClient := pb.NewRuntimeServiceClient(clientConnection)

	cr := &CrioRuntime{
		client: runtimeClient,
	}

	return cr, nil
}
