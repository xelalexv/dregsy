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
func TestE2EDocker(t *testing.T) {
	tryConfig(test.NewTestHelper(t), "e2e/docker.yaml", true, test.GetParams())
}

//
func TestE2EDockerECR(t *testing.T) {
	registries.SkipIfECRNotConfigured(t)
	p := test.GetParams()
	registries.RemoveECRRepo(t, p)
	tryConfig(test.NewTestHelper(t), "e2e/docker-ecr.yaml", true, p)
	registries.RemoveECRRepo(t, p)
}

//
func TestE2EDockerGCR(t *testing.T) {
	registries.SkipIfGCRNotConfigured(t)
	p := test.GetParams()
	registries.RemoveGCRRepo(t, p)
	tryConfig(test.NewTestHelper(t), "e2e/docker-gcr.yaml", true, p)
	registries.RemoveGCRRepo(t, p)
}

//
func TestE2ESkopeo(t *testing.T) {
	tryConfig(test.NewTestHelper(t), "e2e/skopeo.yaml", true, test.GetParams())
}

//
func TestE2ESkopeoECR(t *testing.T) {
	registries.SkipIfECRNotConfigured(t)
	p := test.GetParams()
	registries.RemoveECRRepo(t, p)
	tryConfig(test.NewTestHelper(t), "e2e/skopeo-ecr.yaml", true, p)
	registries.RemoveECRRepo(t, p)
}

//
func TestE2ESkopeoGCR(t *testing.T) {
	registries.SkipIfGCRNotConfigured(t)
	p := test.GetParams()
	registries.RemoveGCRRepo(t, p)
	tryConfig(test.NewTestHelper(t), "e2e/skopeo-gcr.yaml", true, p)
	registries.RemoveGCRRepo(t, p)
}

//
func tryConfig(th *test.TestHelper, file string, verify bool, data interface{}) {

	test.StackTraceDepth = 2
	defer func() { test.StackTraceDepth = 1 }()

	src := th.GetFixture(file)
	dst := src

	if data != nil {
		dst = th.GetFixture("e2e/_dregsy-run.yaml")
		th.AssertNoError(prepareConfig(src, dst, data))
		defer os.Remove(dst)
	}

	th.AssertEqual(0, runDregsy(th, 1, 0, "-config="+dst))

	if !verify {
		return
	}

	log.Info("TEST - validating result")
	c, err := sync.LoadConfig(dst)
	th.AssertNoError(err)

	for _, t := range c.Tasks {
		for _, m := range t.Mappings {
			ref := fmt.Sprintf("%s%s", t.Target.Registry, m.To)
			th.AssertNoError(t.Target.RefreshAuth())
			tags, err := skopeo.ListAllTags(ref,
				skopeo.DecodeJSONAuth(t.Target.Auth), "", t.Target.SkipTLSVerify)
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

	for i := 0; i < 10; i++ {
		select {
		case instance = <-testSync:
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

	for i := 0; i < 10; i++ {
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
