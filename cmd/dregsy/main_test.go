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

package main

import (
	"fmt"
	"os"
	"testing"
	"text/template"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/xelalexv/dregsy/internal/pkg/relays/skopeo"
	"github.com/xelalexv/dregsy/internal/pkg/sync"
	"github.com/xelalexv/dregsy/internal/pkg/test"
	"github.com/xelalexv/dregsy/internal/pkg/test/registries"
)

//
func TestE2EOneoff(t *testing.T) {
	tryConfig(test.NewTestHelper(t), "e2e/base/oneoff.yaml",
		0, 0, true, nil, test.GetParams())
}

//
func TestE2EDocker(t *testing.T) {
	tryConfig(test.NewTestHelper(t), "e2e/base/docker.yaml",
		1, 0, true, nil, test.GetParams())
}

//
func TestE2EDockerECR(t *testing.T) {
	registries.SkipIfECRNotConfigured(t)
	p := test.GetParams()
	registries.RemoveECRRepo(t, p)
	tryConfig(test.NewTestHelper(t), "e2e/base/docker-ecr.yaml",
		1, 0, true, nil, p)
	registries.RemoveECRRepo(t, p)
}

//
func TestE2EDockerGCR(t *testing.T) {
	registries.SkipIfGCPNotConfigured(t, true, false)
	p := test.GetParams()
	registries.EmptyGCRRepo(t, p)
	tryConfig(test.NewTestHelper(t), "e2e/base/docker-gcr.yaml",
		1, 0, true, nil, p)
	registries.EmptyGCRRepo(t, p)
}

//
func TestE2EDockerGCRNoAuth(t *testing.T) {
	tryConfig(test.NewTestHelper(t), "e2e/base/docker-gcr-noauth.yaml",
		1, 0, true, nil, test.GetParams())
}

//
func TestE2EDockerGAR(t *testing.T) {
	registries.SkipIfGCPNotConfigured(t, false, true)
	p := test.GetParams()
	registries.EmptyGARRepo(t, p)
	tryConfig(test.NewTestHelper(t), "e2e/base/docker-gar.yaml",
		1, 0, true, nil, p)
	registries.EmptyGARRepo(t, p)
}

//
func TestE2EDockerGARNoAuth(t *testing.T) {
	tryConfig(test.NewTestHelper(t), "e2e/base/docker-gar-noauth.yaml",
		1, 0, true, nil, test.GetParams())
}

//
func TestE2ESkopeo(t *testing.T) {
	tryConfig(test.NewTestHelper(t), "e2e/base/skopeo.yaml",
		1, 0, true, nil, test.GetParams())
}

//
func TestE2ESkopeoECR(t *testing.T) {
	registries.SkipIfECRNotConfigured(t)
	p := test.GetParams()
	registries.RemoveECRRepo(t, p)
	tryConfig(test.NewTestHelper(t), "e2e/base/skopeo-ecr.yaml",
		1, 0, true, nil, p)
	registries.RemoveECRRepo(t, p)
}

//
func TestE2ESkopeoGCR(t *testing.T) {
	registries.SkipIfGCPNotConfigured(t, true, false)
	p := test.GetParams()
	registries.EmptyGCRRepo(t, p)
	tryConfig(test.NewTestHelper(t), "e2e/base/skopeo-gcr.yaml",
		1, 0, true, nil, p)
	registries.EmptyGCRRepo(t, p)
}

//
func TestE2ESkopeoGCRNoAuth(t *testing.T) {
	tryConfig(test.NewTestHelper(t), "e2e/base/skopeo-gcr-noauth.yaml",
		1, 0, true, nil, test.GetParams())
}

//
func TestE2ESkopeoGAR(t *testing.T) {
	registries.SkipIfGCPNotConfigured(t, false, true)
	p := test.GetParams()
	registries.EmptyGARRepo(t, p)
	tryConfig(test.NewTestHelper(t), "e2e/base/skopeo-gar.yaml",
		1, 0, true, nil, p)
	registries.EmptyGARRepo(t, p)
}

//
func TestE2ESkopeoGARNoAuth(t *testing.T) {
	tryConfig(test.NewTestHelper(t), "e2e/base/skopeo-gar-noauth.yaml",
		1, 0, true, nil, test.GetParams())
}

//
func TestE2EDockerMappingDockerhub(t *testing.T) {
	tryConfig(test.NewTestHelper(t), "e2e/mapping/docker-dockerhub.yaml",
		0, 0, true, map[string][]string{
			"mapping-docker/dh/xelalex/dregsy-dummy-public":  {"latest"},
			"mapping-docker/dh/xelalex/dregsy-dummy-private": {"latest"},
		},
		test.GetParams())
}

//
func TestE2EDockerMappingLocal(t *testing.T) {
	tryConfig(test.NewTestHelper(t), "e2e/mapping/docker-local.yaml",
		0, 0, true, map[string][]string{
			"mapping-docker/dh-copy/xelalex/dregsy-dummy-public":  {"latest"},
			"mapping-docker/dh-copy/xelalex/dregsy-dummy-private": {"latest"},
		},
		test.GetParams())
}

//
func TestE2EDockerMappingDockerhubSearch(t *testing.T) {
	tryConfig(test.NewTestHelper(t), "e2e/mapping/docker-dh-search.yaml",
		0, 0, true, map[string][]string{
			"mapping-docker/dh/other-jenkins/jnlp-slave": {"latest"},
		},
		test.GetParams())
}

//
func TestE2EDockerMappingECR(t *testing.T) {
	registries.SkipIfECRNotConfigured(t)
	tryConfig(test.NewTestHelper(t), "e2e/mapping/docker-ecr.yaml",
		0, 0, true, map[string][]string{
			"mapping-docker/ecr/kubika/brucket":       {"v0.0.1"},
			"mapping-docker/ecr/kubika/brucket-shell": {"v0.0.1"},
		},
		test.GetParams())
}

//
func TestE2ESkopeoMappingDockerhub(t *testing.T) {
	tryConfig(test.NewTestHelper(t), "e2e/mapping/skopeo-dockerhub.yaml",
		0, 0, true, map[string][]string{
			"mapping-skopeo/dh/xelalex/dregsy-dummy-public":  {"latest"},
			"mapping-skopeo/dh/xelalex/dregsy-dummy-private": {"latest"},
		},
		test.GetParams())
}

//
func TestE2ESkopeoMappingLocal(t *testing.T) {
	tryConfig(test.NewTestHelper(t), "e2e/mapping/skopeo-local.yaml",
		0, 0, true, map[string][]string{
			"mapping-skopeo/dh-copy/xelalex/dregsy-dummy-public":  {"latest"},
			"mapping-skopeo/dh-copy/xelalex/dregsy-dummy-private": {"latest"},
		},
		test.GetParams())
}

//
func TestE2ESkopeoMappingDockerhubSearch(t *testing.T) {
	tryConfig(test.NewTestHelper(t), "e2e/mapping/skopeo-dh-search.yaml",
		0, 0, true, map[string][]string{
			"mapping-skopeo/dh/other-jenkins/jnlp-slave": {"latest"},
		},
		test.GetParams())
}

//
func TestE2ESkopeoMappingECR(t *testing.T) {
	registries.SkipIfECRNotConfigured(t)
	tryConfig(test.NewTestHelper(t), "e2e/mapping/skopeo-ecr.yaml",
		0, 0, true, map[string][]string{
			"mapping-skopeo/ecr/kubika/brucket":       {"v0.0.1"},
			"mapping-skopeo/ecr/kubika/brucket-shell": {"v0.0.1"},
		},
		test.GetParams())
}

//
func tryConfig(th *test.TestHelper, file string, ticks int, wait time.Duration,
	verify bool, expectations map[string][]string, data interface{}) {

	test.StackTraceDepth = 2
	defer func() { test.StackTraceDepth = 1 }()

	src := th.GetFixture(file)
	dst := src

	if data != nil {
		dst = th.GetFixture("e2e/_dregsy-run.yaml")
		th.AssertNoError(prepareConfig(src, dst, data))
		defer os.Remove(dst)
	}

	th.AssertEqual(0, runDregsy(th, ticks, wait, "-config="+dst))

	if !verify {
		return
	}

	log.Info("TEST - validating result")
	c, err := sync.LoadConfig(dst)
	th.AssertNoError(err)

	if expectations != nil {
		validateAgainstExpectations(th, c, expectations)
	} else {
		validateAgainstTaskMapping(th, c)
	}
}

//
func validateAgainstExpectations(th *test.TestHelper, c *sync.SyncConfig,
	expectations map[string][]string) {

	if len(c.Tasks) == 0 {
		return
	}

	t := c.Tasks[0]
	th.AssertNoError(t.Target.RefreshAuth())

	for eRef, eTags := range expectations {
		ref := fmt.Sprintf("%s/%s", t.Target.Registry, eRef)
		tags, err := skopeo.ListAllTags(ref,
			skopeo.DecodeJSONAuth(t.Target.GetAuth()),
			"", t.Target.SkipTLSVerify)
		th.AssertNoError(err)
		th.AssertEquivalentSlices(eTags, tags)
	}
}

//
func validateAgainstTaskMapping(th *test.TestHelper, c *sync.SyncConfig) {

	for _, t := range c.Tasks {
		th.AssertNoError(t.Target.RefreshAuth())
		for _, m := range t.Mappings {
			ref := fmt.Sprintf("%s%s", t.Target.Registry, m.To)
			tags, err := skopeo.ListAllTags(ref,
				skopeo.DecodeJSONAuth(t.Target.GetAuth()),
				"", t.Target.SkipTLSVerify)
			th.AssertNoError(err)
			th.AssertEquivalentSlices(m.Tags, tags)
		}
	}
}

//
func prepareConfig(src, dst string, data interface{}) error {

	tmpl, err := template.ParseFiles(src)
	if err != nil {
		return err
	}

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	return tmpl.Execute(out, data)
}

//
func runDregsy(th *test.TestHelper, ticks int, wait time.Duration,
	args ...string) int {

	testRound = true
	testArgs = args
	testSync = make(chan *sync.Sync)
	defer close(testSync)

	go func() {
		main()
		testSync <- nil
	}()

	var instance *sync.Sync

	for i := 10; i > 0; i-- {
		select {
		case instance = <-testSync:
			i = 0
			break
		default:
			time.Sleep(time.Second)
		}
	}

	if instance == nil {
		panic("dregsy did not start")
	}

	for i := ticks; i > 0; i-- {
		instance.WaitForTick()
	}

	if wait > 0 {
		time.Sleep(time.Second * wait)
	}

	if ticks > 0 || wait > 0 {
		log.Info("TEST - shutting down dregsy")
		instance.Shutdown()
	}

	for i := 0; i < 120; i++ {
		select {
		case <-testSync:
			log.Info("TEST - dregsy stopped")
			return dregsyExitCode
		default:
			time.Sleep(time.Second)
		}
	}

	panic("dregsy did not stop")
}
