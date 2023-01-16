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

func (c *cniPodNetwork) ToProperty() cniPodNetworkProperty {
	return cniPodNetworkProperty{
		IfMAC:        c.IfMAC,
		IsPrimary:    c.IsPrimary,
		PodagentSkip: c.PodagentSkip,
	}
}

type cniPodNetworkProperty struct {
	IfMAC     string
	IsPrimary bool
	// PodagentSkip makes podagent skip configuring this network
	PodagentSkip bool
}

type cniPodNetworks []cniPodNetwork

const (
	maxRetries = 60
	retryDelay = 10 * time.Second
)

// Process will take a element from the FIFO queue and attempt to process it (either add or remove network)
func (c *Controller) Process(e *Event) {
	var err error

	attachmentTuple := e.data.(*cni.AttachmentTuple)
	key := c.configStore.getConfigRecordKey(attachmentTuple.PodName, attachmentTuple.NetworkName)
	cfgRecord, err := c.configStore.getConfigRecord(key)
	if err != nil {
		glog.V(3).Infof("network config record not found, ignoring event %+v ", e.data)
		return
	}

	switch cfgRecord.Expected.Optype {
	case Add:
		if cfgRecord.Running.State == Nil {
			err = c.applyAddNetwork(key, cfgRecord, e)
			if err != nil {
				c.eventQueue.Enqueue(&Event{data: e.data})
			}
			return
		}
		if cfgRecord.Running.State == Dirty ||
			!c.configStore.isConfigSame(cfgRecord.Expected, cfgRecord.Running) {
			err = c.applyDeleteNetwork(key, cfgRecord, e)
			if err != nil {
				c.eventQueue.Enqueue(&Event{data: e.data})
				return
			}
			err = c.applyAddNetwork(key, cfgRecord, e)
			if err != nil {
				c.eventQueue.Enqueue(&Event{data: e.data})
			}
			return
		}
		glog.V(3).Infof("ignoring adding network as the same network is already running %+v", e.data)
	case Delete:
		if cfgRecord.Running.State == Nil {
			glog.V(3).Infof("ignoring deleting network as it's not added %+v", e.data)
			return
		}
		err = c.applyDeleteNetwork(key, cfgRecord, e)
		if err != nil {
			c.eventQueue.Enqueue(&Event{data: e.data})
		}
	default:
		glog.Errorf("processing invalid expected state in config record %+v in event %+v", cfgRecord, e)
	}
}

func (c *Controller) applyDeleteNetwork(key string, cfgRecord ConfigRecord, e *Event) error {
	cfgRecord.Running.State = Dirty
	err := c.configStore.saveRunningConfig(key, cfgRecord.Running)
	if err != nil {
		glog.Errorf("Failed saving running config err:%v", err)
		return err
	}

	err = c.cniPlugin.DeleteNetwork(getRunningCNIParams(cfgRecord.Running.Data))
	if err != nil {
		glog.Errorf("Failed deleting network %+v err:%v", e.data, err)
		return fmt.Errorf("Failed to delete network %+v err:%w", e.data, err)
	}
	err = c.configStore.saveRunningConfig(key, RunningConfig{State: Nil})
	if err != nil {
		glog.Errorf("Failed saving running config err:%v", err)
		return err
	}
	glog.V(3).Infof("Succeeded deleting network %+v", e.data)
	return nil
}

func (c *Controller) applyAddNetwork(key string, cfgRecord ConfigRecord, e *Event) error {
	cfgRecord.Running.Data = cfgRecord.Expected.Data
	cfgRecord.Running.State = Dirty
	// Note: saveRunningConfig can fail if the pod is deleted in between,
	// an error is returned to the caller, the caller(worker) requeue the event e again.
	// worker while processing the event e in the next run removes the event permanently.
	err := c.configStore.saveRunningConfig(key, cfgRecord.Running)
	if err != nil {
		glog.Errorf("Failed saving running config err:%v", err)
		return err
	}

	err = c.cniPlugin.AddNetwork(getRunningCNIParams(cfgRecord.Running.Data))
	if err != nil {
		glog.Errorf("Failed adding network %+v err:%v", e.data, err)
		return fmt.Errorf("Failed to add network %+v err:%w", e.data, err)
	}

	cfgRecord.Running.State = Active
	c.configStore.saveRunningConfig(key, cfgRecord.Running)
	if err != nil {
		glog.Errorf("Failed saving running config err:%v", err)
		return err
	}
	glog.V(3).Infof("Succeeded adding network %+v", e.data)
	return nil
}

func getNetworkSet(networks string) (gset.GSet, error) {
	nets := cniPodNetworks{}
	netSetBuilder := gset.NewBuilder()
	err := json.Unmarshal([]byte(networks), &nets)
	if err == nil {
		for _, n := range nets {
			np := n.ToProperty()
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

func getRunningCNIParams(data interface{}) *cni.Parameters {
	tdata := data.(map[string]interface{})
	cniParams := &cni.Parameters{
		Namespace:   tdata["Namespace"].(string),
		PodName:     tdata["PodName"].(string),
		SandboxID:   tdata["SandboxID"].(string),
		NetnsPath:   tdata["NetnsPath"].(string),
		NetworkName: tdata["NetworkName"].(string),
		IfMAC:       tdata["IfMAC"].(string),
	}
	return cniParams
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

func (c *Controller) getCNIAttachmentTuple(podName, networkName string) *cni.AttachmentTuple {
	cniAttachmentTuple := &cni.AttachmentTuple{
		PodName:     podName,
		NetworkName: networkName,
	}
	return cniAttachmentTuple
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
	if np.PodagentSkip {
		glog.V(3).Infof("Skipping adding pod's %s network %s", podObj.GetName(), networkName)
		return nil
	}

	cniParams, err := c.getCNIParams(podObj, networkName, np)
	if err != nil {
		return err
	}

	key := c.configStore.getConfigRecordKey(podObj.GetName(), networkName)
	err = c.configStore.saveExpectedConfig(key, ExpectedConfig{Optype: Add, Data: cniParams})
	if err != nil {
		return err
	}
	c.eventQueue.Enqueue(&Event{data: c.getCNIAttachmentTuple(podObj.GetName(), networkName)})

	return nil
}

func (c *Controller) delNetwork(podObj *apiv1.Pod, networkName string, np cniPodNetworkProperty) error {
	// filter primary networks (i.e. in case we overwrite the default network attatchement on eth0)
	if np.IsPrimary {
		return nil
	}
	if np.PodagentSkip {
		glog.V(3).Infof("Skipping deleting pod's %s network %s", podObj.GetName(), networkName)
		return nil
	}

	key := c.configStore.getConfigRecordKey(podObj.GetName(), networkName)
	err := c.configStore.saveExpectedConfig(key, ExpectedConfig{Optype: Delete})
	if err != nil {
		return err
	}
	c.eventQueue.Enqueue(&Event{data: c.getCNIAttachmentTuple(podObj.GetName(), networkName)})
	return nil
}

func (c *Controller) podAdded(podObj interface{}) {
	pod := podObj.(*apiv1.Pod)
	podName := pod.ObjectMeta.Name
	glog.V(5).Infof("Pod added: %s", podName)
	networks, ok := pod.Annotations["networks"]
	if !ok {
		glog.V(5).Infof("Pod %s does not contain any networks, skipping", podName)
		return
	}

	c.addNetworks(pod, networks)
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
			for _, n := range nets {
				np := n.ToProperty()
				err := c.delNetwork(newPod, n.NetworkName, np)
				if err != nil {
					glog.Errorf("Failed to delete network %s on pod %s", n.NetworkName, podName)
				}
			}
		}
	} else if newNetworks, ok := newPod.Annotations["networks"]; ok {
		c.addNetworks(newPod, newNetworks)
	}
}

func (c *Controller) podDeleted(podObj interface{}) {
	pod := podObj.(*apiv1.Pod)
	podName := pod.ObjectMeta.Name
	glog.V(5).Infof("Pod Deleted: %s", podName)
	networks, ok := pod.Annotations["networks"]
	if !ok {
		glog.V(5).Infof("Pod %s does not contain any networks, skipping", podName)
		return
	}

	c.delPendingNetworks(pod, networks)
}

func (c *Controller) delPendingNetworks(pod *apiv1.Pod, networkAnnotation string) {
	podName := pod.ObjectMeta.Name
	glog.V(5).Infof("Pod's %s with networks annotation '%s' got deleted", podName, networkAnnotation)
	nets, err := getNetworks(networkAnnotation)
	if err != nil {
		glog.V(4).Infof("Failed to unmarshall pod's %s existing networks annotation, ignore: %s", podName, err)
		return
	}

	for _, n := range nets {
		key := c.configStore.getConfigRecordKey(podName, n.NetworkName)
		c.configStore.delConfigRecord(key)
		glog.V(5).Infof("Pod's %s pending network '%s' got deleted", podName, n.NetworkName)
	}
}

func (c *Controller) addNetworks(pod *apiv1.Pod, networkAnnotation string) {
	podName := pod.ObjectMeta.Name
	glog.V(5).Infof("Pod's %s networks annotation '%s' got added", podName, networkAnnotation)
	nets, err := getNetworks(networkAnnotation)
	if err != nil {
		glog.V(4).Infof("Failed to unmarshall pod's %s new networks annotation, ignore: %s", podName, err)
		return
	}

	for _, n := range nets {
		np := n.ToProperty()
		err := c.addNetwork(pod, n.NetworkName, np)
		if err != nil {
			glog.Errorf("Failed to add network %s on pod %s", n.NetworkName, podName)
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
	// Setup an informer to call functions when the watchlist changes
	_, controller := cache.NewInformer(
		watchlist,
		&apiv1.Pod{},
		// we don't use any resync because UpdateFunc logic currently compares
		// the annotations between the old and new objects, but during a resync,
		// old == new so nothing happens
		0,
		cache.ResourceEventHandlerFuncs{
			AddFunc:    c.podAdded,
			UpdateFunc: c.podUpdated,
			DeleteFunc: c.podDeleted,
		},
	)

	//Run the controller as a goroutine
	go controller.Run(ctx.Done())
	return controller, nil
}
