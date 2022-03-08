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

import "testing"

type builder struct {
	value      int
	built      *Notifier
	otherValue int
}

func (b *builder) editValue() {
	go func() {
		b.built.Reset()
		b.makeValueFive()
		b.built.Close()
	}()
}

func (b *builder) makeValueFive() {
	b.value = 5
}

func TestNotifierRace(t *testing.T) {
	b := &builder{
		built: NewNotifier(),
	}
	go func() {
		b.editValue()
		b.built.Wait()
		if b.value > 0 {
			b.otherValue = 3
		}
	}()
}
