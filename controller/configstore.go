/*
Copyright 2017-2023 Kaloom Inc.

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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sync"

	"github.com/golang/glog"
)

const (
	defaultConfigDir = "/var/run/podagent/configstore/"
)

type Optype string

const (
	Add    Optype = "Add"
	Delete Optype = "Delete"
)

// ExpectedConfig struct
type ExpectedConfig struct {
	Optype Optype
	Data   interface{}
}

type RunningState string

const (
	Nil    RunningState = "Nil"
	Active RunningState = "Active"
	Dirty  RunningState = "Dirty"
)

// RunningConfig struct
type RunningConfig struct {
	State RunningState
	Data  interface{}
}

type ConfigRecord struct {
	Expected ExpectedConfig
	Running  RunningConfig
}

// ConfigStore is a store for Config
type ConfigStore struct {
	mu  sync.Mutex
	dir string
}

// newConfigStore will create a new config store
func newConfigStore() *ConfigStore {
	return &ConfigStore{dir: defaultConfigDir}
}

func (cs *ConfigStore) getConfigRecordKey(podName, networkName string) string {
	return fmt.Sprintf("%s-%s.json", podName, networkName)
}

func (cs *ConfigStore) isConfigSame(expected ExpectedConfig, running RunningConfig) bool {
	return reflect.DeepEqual(expected.Data, running.Data)
}

func (cs *ConfigStore) saveRunningConfig(key string, running RunningConfig) error {
	glog.V(3).Infof("Saving running config:%+v, with Key: %s", running, key)
	cs.mu.Lock()
	defer cs.mu.Unlock()
	var currConfigRec ConfigRecord
	path := filepath.Join(cs.dir, key)
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read config data from the path(%q): %w", path, err)
	}

	if !json.Valid([]byte(data)) {
		return fmt.Errorf("invalid JSON string fetched from the path(%q): %s", path, data)
	}

	err = json.Unmarshal(data, &currConfigRec)
	if err != nil {
		return fmt.Errorf("error unmarshalling config data from the path(%q): %w", path, err)
	}

	currConfigRec.Running = running
	configRecBytes, err := json.Marshal(currConfigRec)
	if err != nil {
		return fmt.Errorf("error serializing config: %v", err)
	}
	err = os.WriteFile(path, configRecBytes, 0600)
	if err != nil {
		return fmt.Errorf("failed to write config data in the path(%q): %w", path, err)
	}

	return err
}

func (cs *ConfigStore) saveExpectedConfig(key string, expected ExpectedConfig) error {
	glog.V(3).Infof("Saving expected config:%+v, with Key: %s", expected, key)
	cs.mu.Lock()
	defer cs.mu.Unlock()
	currConfigRec := ConfigRecord{
		Running: RunningConfig{State: Nil},
	}

	if err := os.MkdirAll(cs.dir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory(%q): %w", cs.dir, err)
	}

	path := filepath.Join(cs.dir, key)
	data, err := os.ReadFile(path)
	if err == nil { //config record does exist
		if !json.Valid([]byte(data)) {
			return fmt.Errorf("invalid JSON string fetched from the path(%q): %s", path, data)
		}
		err = json.Unmarshal(data, &currConfigRec)
		if err != nil {
			return fmt.Errorf("error unmarshalling config data from the path(%q): %w", path, err)
		}
	}

	currConfigRec.Expected = expected
	configRecBytes, err := json.Marshal(currConfigRec)
	if err != nil {
		return fmt.Errorf("error serializing config: %v", err)
	}
	err = os.WriteFile(path, configRecBytes, 0600)
	if err != nil {
		return fmt.Errorf("failed to write config data in the path(%q): %w", path, err)
	}

	return nil
}

func (cs *ConfigStore) getConfigRecord(key string) (ConfigRecord, error) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	path := filepath.Join(cs.dir, key)
	data, err := os.ReadFile(path)
	if err != nil {
		return ConfigRecord{}, fmt.Errorf("failed to read config data from the path(%q): %w", path, err)
	}

	if !json.Valid([]byte(data)) {
		return ConfigRecord{}, fmt.Errorf("Invalid JSON string fetched from the path(%q): %s", path, data)
	}

	var currConfigRec ConfigRecord
	err = json.Unmarshal(data, &currConfigRec)
	if err != nil {
		return ConfigRecord{}, fmt.Errorf("error unmarshalling config data from the path(%q): %w", path, err)
	}

	glog.V(3).Infof("Returning configRecord:%+v, with Key: %s", currConfigRec, key)
	return currConfigRec, nil
}

func (cs *ConfigStore) delConfigRecord(key string) error {
	glog.V(3).Infof("Deleting configRecord with Key: %s", key)
	cs.mu.Lock()
	defer cs.mu.Unlock()
	path := filepath.Join(cs.dir, key)
	return os.Remove(path)
}
