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

import (
	"math/rand"
	"testing"
	"testing/quick"
	"time"
)

func TestNotifier(t *testing.T) {
	err := quick.Check(func() bool {
		rand.Seed(time.Now().UnixNano())
		type foo struct {
			value   int
			created *Notifier
		}

		f := foo{
			created: NewNotifier(),
			value:   3,
		}
		f.value = 3
		go func() {
			time.Sleep(time.Duration(rand.Intn(100) * int(time.Millisecond)))
			f.value = 5
			f.created.Close()
		}()
		f.created.Wait()
		if f.value != 5 {
			return false
		}
		f.created.Reset()
		go func() {
			time.Sleep(time.Duration(rand.Intn(100) * int(time.Millisecond)))
			f.value = 6
			f.created.Close()
		}()
		f.created.Wait()
		return f.value == 6

	}, &quick.Config{
		MaxCount: 100,
	})

	if err != nil {
		t.Error("expecting a value we had to wait for, but did not get it")
	}

}
