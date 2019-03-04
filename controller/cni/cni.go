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

package cni

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/containernetworking/cni/libcni"
	cnitypes "github.com/containernetworking/cni/pkg/types"
	"github.com/golang/glog"
	"k8s.io/kubernetes/pkg/kubelet/network"
	utilexec "k8s.io/utils/exec"
)

const (
	// DefaultNetDir default directory for the cni config
	DefaultNetDir = "/etc/cni/net.d"
	// DefaultCNIDir default directory for the cni binary
	DefaultCNIDir = "/opt/cni/bin"
)

// Parameters the params struct for the cni-plugin
type Parameters struct {
	Namespace   string
	PodName     string
	SandboxID   string
	NetnsPath   string
	NetworkName string
	IfMAC       string
}

// NetworkPlugin object to export
type NetworkPlugin struct {
	sync.RWMutex
	defaultNetwork *cniNetwork

	execer      utilexec.Interface
	nsenterPath string
	pluginDir   string
	binDir      string
	vendorName  string
}

type cniNetwork struct {
	name          string
	NetworkConfig *libcni.NetworkConfigList
	CNIConfig     libcni.CNI
}

func getDefaultCNINetwork(pluginDir, binDir, vendorName string) (*cniNetwork, error) {
	if pluginDir == "" {
		pluginDir = DefaultNetDir
	}
	files, err := libcni.ConfFiles(pluginDir, []string{".conf", ".conflist", ".json"})
	switch {
	case err != nil:
		return nil, err
	case len(files) == 0:
		return nil, fmt.Errorf("No networks found in %s", pluginDir)
	}

	sort.Strings(files)
	for _, confFile := range files {
		var confList *libcni.NetworkConfigList
		if strings.HasSuffix(confFile, ".conflist") {
			confList, err = libcni.ConfListFromFile(confFile)
			if err != nil {
				glog.Warningf("Error loading CNI config list file %s: %v", confFile, err)
				continue
			}
		} else {
			conf, err := libcni.ConfFromFile(confFile)
			if err != nil {
				glog.Warningf("Error loading CNI config file %s: %v", confFile, err)
				continue
			}
			// Ensure the config has a "type" so we know what plugin to run.
			// Also catches the case where somebody put a conflist into a conf file.
			if conf.Network.Type == "" {
				glog.Warningf("Error loading CNI config file %s: no 'type'; perhaps this is a .conflist?", confFile)
				continue
			}

			confList, err = libcni.ConfListFromConf(conf)
			if err != nil {
				glog.Warningf("Error converting CNI config file %s to list: %v", confFile, err)
				continue
			}
		}
		if len(confList.Plugins) == 0 {
			glog.Warningf("CNI config list %s has no networks, skipping", confFile)
			continue
		}
		confType := confList.Plugins[0].Network.Type

		// Search for vendor-specific plugins as well as default plugins in the CNI codebase.
		vendorDir := vendorCNIDir(vendorName, confType)
		cninet := &libcni.CNIConfig{
			Path: []string{vendorDir, binDir},
		}
		network := &cniNetwork{name: confList.Name, NetworkConfig: confList, CNIConfig: cninet}
		return network, nil
	}
	return nil, fmt.Errorf("No valid networks found in %s", pluginDir)
}

func vendorCNIDir(vendorName, pluginType string) string {
	if vendorName != "" {
		return fmt.Sprintf("/opt/%s/cni/bin", vendorName)
	}
	return fmt.Sprintf("/opt/%s/bin", pluginType)
}

// NewCNIPlugin instantiate a cni plugin object
func NewCNIPlugin(cniBinPath, cniConfPath, cniVendorName string) (*NetworkPlugin, error) {
	var err error
	plugin := &NetworkPlugin{
		binDir:     cniBinPath,
		pluginDir:  cniConfPath,
		vendorName: cniVendorName,
		execer:     utilexec.New(),
	}
	plugin.nsenterPath, err = plugin.execer.LookPath("nsenter")
	if err != nil {
		return nil, err
	}

	plugin.syncNetworkConfig()
	return plugin, nil
}

func (plugin *NetworkPlugin) syncNetworkConfig() {
	network, err := getDefaultCNINetwork(plugin.pluginDir, plugin.binDir, plugin.vendorName)
	if err != nil {
		glog.Warningf("Unable to update cni config: %s", err)
		return
	}
	plugin.setDefaultNetwork(network)
}

func (plugin *NetworkPlugin) getDefaultNetwork() *cniNetwork {
	plugin.RLock()
	defer plugin.RUnlock()
	return plugin.defaultNetwork
}

func (plugin *NetworkPlugin) setDefaultNetwork(n *cniNetwork) {
	plugin.Lock()
	defer plugin.Unlock()
	plugin.defaultNetwork = n
}

func (plugin *NetworkPlugin) checkInitialized() error {
	if plugin.getDefaultNetwork() == nil {
		return errors.New("cni config uninitialized")
	}
	return nil
}

// AddNetwork add a network attachment off cniParams
func (plugin *NetworkPlugin) AddNetwork(cniParams *Parameters) error {
	if err := plugin.checkInitialized(); err != nil {
		return err
	}
	_, err := plugin.addToNetwork(plugin.getDefaultNetwork(), cniParams)
	if err != nil {
		glog.Errorf("Error while adding to cni network: %s", err)
		return err
	}

	return err
}

// DeleteNetwork delete a network attachment off cniParams
func (plugin *NetworkPlugin) DeleteNetwork(cniParams *Parameters) error {
	if err := plugin.checkInitialized(); err != nil {
		return err
	}
	return plugin.deleteFromNetwork(plugin.getDefaultNetwork(), cniParams)
}

func (plugin *NetworkPlugin) addToNetwork(network *cniNetwork, cniParams *Parameters) (cnitypes.Result, error) {
	rt, err := plugin.buildCNIRuntimeConf(cniParams)
	if err != nil {
		glog.Errorf("Error adding network when building cni runtime conf: %v", err)
		return nil, err
	}

	netConf, cniNet := network.NetworkConfig, network.CNIConfig
	glog.V(4).Infof("About to add CNI network %v (type=%v)", cniParams.NetworkName, netConf.Plugins[0].Network.Type)
	res, err := cniNet.AddNetworkList(netConf, rt)
	if err != nil {
		glog.Errorf("Error adding network: %v", err)
		return nil, err
	}

	return res, nil
}

func (plugin *NetworkPlugin) deleteFromNetwork(network *cniNetwork, cniParams *Parameters) error {
	rt, err := plugin.buildCNIRuntimeConf(cniParams)
	if err != nil {
		glog.Errorf("Error deleting network when building cni runtime conf: %v", err)
		return err
	}

	netConf, cniNet := network.NetworkConfig, network.CNIConfig
	glog.V(4).Infof("About to del CNI network %v (type=%v)", cniParams.NetworkName, netConf.Plugins[0].Network.Type)
	err = cniNet.DelNetworkList(netConf, rt)
	if err != nil {
		glog.Errorf("Error deleting network: %v", err)
		return err
	}
	return nil
}

func (plugin *NetworkPlugin) buildCNIRuntimeConf(cniParams *Parameters) (*libcni.RuntimeConf, error) {
	glog.V(4).Infof("Pod's %s cni parameters: netns path %s in namespace %s", cniParams.PodName, cniParams.NetnsPath, cniParams.Namespace)

	rt := &libcni.RuntimeConf{
		ContainerID: cniParams.SandboxID,
		NetNS:       cniParams.NetnsPath,
		IfName:      network.DefaultInterfaceName,
		Args: [][2]string{
			{"IgnoreUnknown", "1"},
			{"K8S_POD_NAMESPACE", cniParams.Namespace},
			{"K8S_POD_NAME", cniParams.PodName},
			{"K8S_POD_INFRA_CONTAINER_ID", cniParams.SandboxID},
			{"K8S_POD_NETWORK", cniParams.NetworkName},
			{"K8S_POD_IFMAC", cniParams.IfMAC},
		},
	}

	return rt, nil
}
