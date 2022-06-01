/*
	Copyright 2020 Alexander Vollschwitz <xelalex@gmx.net>

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

package sync

import (
	"testing"

	"github.com/xelalexv/dregsy/internal/pkg/test"
)

//
func TestInvalidSync(t *testing.T) {

	th := test.NewTestHelper(t)

	// mappings
	trySync(th, "config/docker-platform-all.yaml",
		"relay 'docker' does not support mappings with 'platform: all'")
}

//
func trySync(th *test.TestHelper, file, err string) (*Sync, error) {

	test.StackTraceDepth = 2
	defer func() { test.StackTraceDepth = 1 }()

	c, e := LoadConfig(th.GetFixture(file))
	th.AssertNoError(e)
	th.AssertNotNil(c)

	s, e := New(c)
	if s != nil {
		defer func() { s.Dispose() }()
	}

	if err != "" {
		th.AssertError(e, err)
		th.AssertNil(s)
	} else {
		th.AssertNoError(e)
		th.AssertNotNil(s)
	}

	return s, e
}
