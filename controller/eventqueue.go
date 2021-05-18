/*
Copyright 2017-2021 Kaloom Inc.

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
	"container/list"
	"fmt"
	"github.com/golang/glog"
	"sync"
)

type OpType int

const (
	Add OpType = iota
	Delete
)

// Event struct
type Event struct {
	opType OpType
	data   interface{}
}

// EventQueue is a FIFO type queue
type EventQueue struct {
	q    *list.List
	m    map[string]*list.Element // ref in the eventQueue list
	lock sync.Mutex
	cond *sync.Cond
}

// newQueue will create a new FIFO queue
func newQueue() *EventQueue {
	eq := &EventQueue{m: make(map[string]*list.Element), q: list.New()}
	eq.q.Init()
	eq.cond = sync.NewCond(&eq.lock)
	return eq
}

// getKey will create a unique key identifying the event
func (ev *Event) getKey() string {
	return fmt.Sprintf("%+v", ev.data)
}

// Enqueue will push the new event in the FIFO queue
// If a similar event already exists with a opposite operation type, both event will be discarded.
func (eq *EventQueue) Enqueue(event *Event) {
	eq.cond.L.Lock()
	defer eq.cond.L.Unlock()

	key := event.getKey()
	glog.V(5).Infof("Enqueuing using key:%s", key)
	if e, ok := eq.m[key]; ok {
		ev := e.Value.(Event)
		if ev.opType != event.opType {
			glog.Infof("Cancelling events from queue - events cancels each others:", ev)
			eq.q.Remove(e)
			delete(eq.m, key)
			return
		}
	}

	e := eq.q.PushBack(*event)
	glog.V(5).Infof("Enqueue new event:", event)
	eq.m[key] = e
	eq.cond.Signal()
}

// Dequeue will remove the first element from the queue and return it for processing.
// The caller MUST use the mutex provided by the EventQueue struct
func (eq *EventQueue) Dequeue() *Event {
	e := eq.q.Front() // First element
	if e == nil {
		return nil
	}
	ev := e.Value.(Event)
	eq.q.Remove(e)
	delete(eq.m, ev.getKey())
	return &ev
}
