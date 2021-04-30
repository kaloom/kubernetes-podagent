/*
Copyright 2017-2019 Kaloom Inc.

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

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	kc "github.com/kaloom/kubernetes-common"
	"github.com/kaloom/kubernetes-common/gset"

	"github.com/kaloom/kubernetes-podagent/controller/cni"

	"github.com/golang/glog"

	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/tools/cache"
)

type cniPodNetwork struct {
	kc.NetworkConfig
}

type cniPodNetworkProperty struct {
	IfMAC     string
	IsPrimary bool
}

type cniPodNetworks []cniPodNetwork

const (
	maxRetries = 60
	retryDelay = 10 * time.Second
)

// Process will take a element from the FIFO queue and attempt to process it (either add or remove network)
func (c *Controller) Process(e *Event) {
	var err error
	switch e.opType {
	case Add:
		for i := 0; i < maxRetries; i++ {
			err = c.cniPlugin.AddNetwork(e.data.(*cni.Parameters))
			if err == nil {
				if i > 0 {
					glog.Infof("Succeeded adding network %+v after %d attempt", e.data, i)
				} else {
					glog.V(5).Infof("Succeeded adding network %+v on first attempt", e.data)
				}
				return
			}
			glog.Warningf("Failed adding network %+v... retrying %d/%d. err:%v", e.data, i+1, maxRetries, err)
			time.Sleep(retryDelay)
		}
		glog.Errorf("Failed adding network %+v after %d attempt. err:%v", e.data, maxRetries, err)

	case Delete:
		for i := 0; i < maxRetries; i++ {
			err = c.cniPlugin.DeleteNetwork(e.data.(*cni.Parameters))
			if err == nil {
				if i > 0 {
					glog.Infof("Succeeded deleting network %+v after %d attempt", e.data, i)
				} else {
					glog.V(5).Infof("Succeeded deleting network %+v on first attempt", e.data)
				}
				return
			}
			glog.Warningf("Failed deleting network %+v... retrying %d/%d. err:%v", e.data, i+1, maxRetries, err)
			time.Sleep(retryDelay)
		}
		glog.Errorf("Failed deleting network %+v after %d attempt. err:%v", e.data, maxRetries, err)

	default:
		glog.Errorf("processing invalid operation type value in event %+v", e)
	}
}

func getNetworkSet(networks string) (gset.GSet, error) {
	nets := cniPodNetworks{}
	netSetBuilder := gset.NewBuilder()
	err := json.Unmarshal([]byte(networks), &nets)
	if err == nil {
		np := cniPodNetworkProperty{}
		for _, n := range nets {
			np.IfMAC = n.IfMAC
			np.IsPrimary = n.IsPrimary
			netSetBuilder.Add(n.NetworkName, np)
		}
	}
	return netSetBuilder.Result(), err
}

func getNetworks(networks string) (cniPodNetworks, error) {
	nets := cniPodNetworks{}
	if err := json.Unmarshal([]byte(networks), &nets); err != nil {
		return nil, err
	}
	return nets, nil
}

func getContainerID(pod *apiv1.Pod) string {
	cidURI := pod.Status.ContainerStatuses[0].ContainerID
	// format is docker://<cid>
	parts := strings.Split(cidURI, "//")
	if len(parts) > 1 {
		return parts[1]
	}
	return cidURI
}

func (c *Controller) getCNIParams(podObj *apiv1.Pod, networkName string, np cniPodNetworkProperty) (*cni.Parameters, error) {
	podName := podObj.ObjectMeta.Name
	namespace := podObj.ObjectMeta.Namespace
	if containerID := getContainerID(podObj); containerID != "" {
		// the sandbox is the "pause" container
		sandboxID, err := c.runtime.GetSandboxID(containerID)
		if err != nil {
			glog.Errorf("Failed to get Pod's %s sandbox ID from cri: %s", podName, err)
			return nil, err
		}
		netns, err := c.runtime.GetNetNS(sandboxID)
		if err != nil {
			glog.Errorf("Failed to get netns of sandbox ID %s: %v", sandboxID, err)
			return nil, err
		}
		cniParams := &cni.Parameters{
			Namespace:   namespace,
			PodName:     podName,
			SandboxID:   sandboxID,
			NetnsPath:   netns,
			NetworkName: networkName,
			IfMAC:       np.IfMAC,
		}
		return cniParams, nil
	}
	return nil, fmt.Errorf("Failed to get Pod's %s container ID", podName)
}

func (c *Controller) addNetwork(podObj *apiv1.Pod, networkName string, np cniPodNetworkProperty) error {
	// filter primary networks (i.e. in case we overwrite the default network attatchement on eth0)
	if np.IsPrimary {
		return nil
	}
	cniParams, err := c.getCNIParams(podObj, networkName, np)
	if err != nil {
		return err
	}

	c.eventQueue.Enqueue(&Event{opType: Add, data: cniParams})
	return nil
}

func (c *Controller) delNetwork(podObj *apiv1.Pod, networkName string, np cniPodNetworkProperty) error {
	// filter primary networks (i.e. in case we overwrite the default network attatchement on eth0)
	if np.IsPrimary {
		return nil
	}
	cniParams, err := c.getCNIParams(podObj, networkName, np)
	if err != nil {
		return err
	}

	c.eventQueue.Enqueue(&Event{opType: Delete, data: cniParams})
	return nil
}

func (c *Controller) podUpdated(oldObj, newObj interface{}) {
	oldPod := oldObj.(*apiv1.Pod)
	newPod := newObj.(*apiv1.Pod)
	podName := oldPod.ObjectMeta.Name
	glog.V(5).Infof("Pod updated: %s", podName)
	if oldNetworks, ok := oldPod.Annotations["networks"]; ok {
		oldNetSet, err := getNetworkSet(oldNetworks)
		if err != nil {
			glog.V(4).Infof("Failed to unmarshall pod's %s old networks annotation, ignore: %s", podName, err)
			return
		}
		if newNetworks, ok := newPod.Annotations["networks"]; ok {
			newNetSet, err := getNetworkSet(newNetworks)
			if err != nil {
				glog.V(4).Infof("Failed to unmarshall pod's %s new networks annotation, ignore: %s", podName, err)
				return
			}
			if d := oldNetSet.Difference(newNetSet); d.Size() > 0 {
				glog.V(5).Infof("The following network(s) got deleted from Pod %s: %s", podName, d)
				for _, netKV := range d.ToSlice() {
					err := c.delNetwork(newPod, netKV.Key, netKV.Val.(cniPodNetworkProperty))
					if err != nil {
						glog.Errorf("Failed to delete network %s on pod %s", netKV.Key, podName)
					}
				}
			}
			if d := newNetSet.Difference(oldNetSet); d.Size() > 0 {
				glog.V(5).Infof("The following network(s) got added to Pod %s: %s", podName, d)
				for _, netKV := range d.ToSlice() {
					err := c.addNetwork(newPod, netKV.Key, netKV.Val.(cniPodNetworkProperty))
					if err != nil {
						glog.Errorf("Failed to add network %s on pod %s", netKV.Key, podName)
					}
				}
			}
		} else {
			glog.V(5).Infof("Pod's %s networks annotation '%s' got deleted", podName, oldNetworks)
			nets, err := getNetworks(oldNetworks)
			if err != nil {
				glog.V(4).Infof("Failed to unmarshall pod's %s old networks annotation, ignore: %s", podName, err)
				return
			}
			np := cniPodNetworkProperty{}
			for _, n := range nets {
				np.IfMAC = n.IfMAC
				np.IsPrimary = n.IsPrimary
				err := c.delNetwork(newPod, n.NetworkName, np)
				if err != nil {
					glog.Errorf("Failed to delete network %s on pod %s", n.NetworkName, podName)
				}
			}
		}
	} else if newNetworks, ok := newPod.Annotations["networks"]; ok {
		glog.V(5).Infof("Pod's %s networks annotation '%s' got added", podName, newNetworks)
		nets, err := getNetworks(newNetworks)
		if err != nil {
			glog.V(4).Infof("Failed to unmarshall pod's %s new networks annotation, ignore: %s", podName, err)
			return
		}
		np := cniPodNetworkProperty{}
		for _, n := range nets {
			np.IfMAC = n.IfMAC
			np.IsPrimary = n.IsPrimary
			err := c.addNetwork(newPod, n.NetworkName, np)
			if err != nil {
				glog.Errorf("Failed to add network %s on pod %s", n.NetworkName, podName)
			}
		}
	}
}

func (c *Controller) eventQueueWorker() {
	for {
		c.eventQueue.cond.L.Lock()

		for c.eventQueue.q.Len() == 0 {
			c.eventQueue.cond.Wait()
		}

		ev := c.eventQueue.Dequeue()
		c.eventQueue.cond.L.Unlock()

		if ev != nil {
			glog.V(5).Infof("Processing event:", ev)
			c.Process(ev)
		}
	}
}

func (c *Controller) watchPods(ctx context.Context, nodeName string) (cache.Controller, error) {

	// Initialize the worker queue
	go c.eventQueueWorker()

	// Currently there is no field selector for a Pod annotation
	// https://github.com/kubernetes/kubernetes/blob/master/pkg/registry/core/pod/strategy.go
	fs := fields.Set{
		"status.phase": "Running",
	}
	if nodeName != "" {
		fs["spec.nodeName"] = nodeName
	}
	fieldsToMatch := fs.AsSelector()
	// Define what we want to look for (Pods)
	watchlist := cache.NewListWatchFromClient(c.kubeClient.CoreV1().RESTClient(), "pods", apiv1.NamespaceAll, fieldsToMatch)
	// batching and collapsing events is controlled by the resyncPeriod, 0 would disable the resync
	resyncPeriod := 30 * time.Second
	// Setup an informer to call functions when the watchlist changes
	_, controller := cache.NewInformer(
		watchlist,
		&apiv1.Pod{},
		resyncPeriod,
		cache.ResourceEventHandlerFuncs{
			UpdateFunc: c.podUpdated,
		},
	)

	//Run the controller as a goroutine
	go controller.Run(ctx.Done())
	return controller, nil
}
