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
func TestValidSyncConfigs(t *testing.T) {

	th := test.NewTestHelper(t)

	c, e := LoadConfig(th.GetFixture("e2e/skopeo.yaml"))
	th.AssertNoError(e)
	th.AssertNotNil(c)
	th.AssertEqual("skopeo", c.Relay)

	c, e = LoadConfig(th.GetFixture("e2e/docker.yaml"))
	th.AssertNoError(e)
	th.AssertNotNil(c)
	th.AssertEqual("docker", c.Relay)
}

//
func TestInvalidSyncConfigs(t *testing.T) {

	th := test.NewTestHelper(t)

	// relay
	tryConfig(th, "config/invalid-relay.yaml", "invalid relay type")
	tryConfig(th, "config/multiple-relays.yaml",
		"setting 'dockerhost' implies 'docker' relay")

	// task
	tryConfig(th, "config/task-no-name.yaml", "a task requires a name")
	tryConfig(th, "config/task-low-interval.yaml",
		"minimum task interval is 30 seconds")
	tryConfig(th, "config/task-bad-interval.yaml",
		"task interval needs to be 0 or a positive integer")
	tryConfig(th, "config/task-no-source.yaml",
		"source registry in task 'test' invalid: location is nil")
	tryConfig(th, "config/task-no-target.yaml",
		"target registry in task 'test' invalid: location is nil")

	// source & target locations
	tryConfig(th, "config/source-no-registry.yaml",
		"source registry in task 'test' invalid: registry not set")
	tryConfig(th, "config/source-not-ecr.yaml", "is not an ECR registry")

	// mappings
	tryConfig(th, "config/mapping-no-from.yaml", "mapping without 'From' path")
}

//
func tryConfig(th *test.TestHelper, file, err string) (*SyncConfig, error) {

	test.StackTraceDepth = 2
	defer func() { test.StackTraceDepth = 1 }()

	c, e := LoadConfig(th.GetFixture(file))
	if err != "" {
		th.AssertError(e, err)
		th.AssertNil(c)
	} else {
		th.AssertNoError(e)
		th.AssertNotNil(c)
	}

	return c, e
}
