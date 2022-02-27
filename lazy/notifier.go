// Copyright 2019 The Hugo Authors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package lazy

import "sync"

// Notifier as a synchronization tool that is queried for just-in-time access
// to a resource. Callers use Wait to block until the resource is ready, and
// call Close to indicate that the resource is ready. Reset returns the
// resource to its locked state.
//
// Notifier must be initialized by calling NewNotifier.
type Notifier struct {
	ch chan struct{}
	// For locking the channel while resetting it
	mu *sync.RWMutex
}

// NewNotifier creates a Notifier with all synchronization mechanisms
// initialized.
func NewNotifier() *Notifier {
	return &Notifier{
		ch: make(chan struct{}),
		mu: &sync.RWMutex{},
	}
}

// Wait waits for the Notifier to be ready, i.e., for Close to be called
// somewhere
func (n *Notifier) Wait() {
	n.mu.RLock()
	defer n.mu.RUnlock()
	<-n.ch
	return
}

// Close unblocks any goroutines that called Wait
func (n *Notifier) Close() {
	n.mu.Lock()
	defer n.mu.Unlock()
	close(n.ch)
	return
}

// Reset returns the resource to its pre-ready state while locking
func (n *Notifier) Reset() {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.ch = make(chan struct{})
	return
}
