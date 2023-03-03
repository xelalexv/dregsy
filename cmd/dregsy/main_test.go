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
	"github.com/xelalexv/dregsy/internal/pkg/util"
)

//
var testPlatforms []string = []string{
	"linux/amd64",
	"linux/386",
	"linux/mips64le",
	"linux/ppc64le",
	"linux/arm/v5",
	"linux/arm/v6",
	"linux/arm/v7",
	"linux/arm64/v8",
}

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
func TestE2EDockerPlatform(t *testing.T) {
	tryConfig(test.NewTestHelper(t), "e2e/base/docker-platform.yaml",
		1, 0, true, nil, test.GetParams())
}

//
func TestE2EDockerECR(t *testing.T) {
	registries.SkipIfECRNotConfigured(t, false)
	p := test.GetParams()
	registries.RemoveECRRepo(t, p, false)
	tryConfig(test.NewTestHelper(t), "e2e/base/docker-ecr.yaml",
		1, 0, true, nil, p)
	registries.RemoveECRRepo(t, p, false)
}

//
func TestE2EDockerECRPub(t *testing.T) {
	registries.SkipIfECRNotConfigured(t, true)
	p := test.GetParams()
	registries.RemoveECRRepo(t, p, true)
	tryConfig(test.NewTestHelper(t), "e2e/base/docker-ecr-pub.yaml",
		1, 0, true, nil, p)
	registries.RemoveECRRepo(t, p, true)
}

//
func TestE2EDockerECRPubNoAuth(t *testing.T) {
	tryConfig(test.NewTestHelper(t), "e2e/base/docker-ecr-pub-noauth.yaml",
		1, 0, true, nil, test.GetParams())
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
	registries.SkipIfECRNotConfigured(t, false)
	tryConfig(test.NewTestHelper(t), "e2e/mapping/docker-ecr.yaml",
		0, 0, true, map[string][]string{
			"mapping-docker/ecr/kubika/brucket":       {"v0.0.1"},
			"mapping-docker/ecr/kubika/brucket-shell": {"v0.0.1"},
		},
		test.GetParams())
}

//
func TestE2EDockerTagSetsRange(t *testing.T) {
	tryConfig(test.NewTestHelper(t), "e2e/tagsets/docker-range.yaml",
		0, 0, true, map[string][]string{
			"tagsets-docker/range/busybox": {
				"latest", "1.31", "1.31.0", "1.31.1-musl", "1.31.1",
				"1.31.1-uclibc", "1.31.1-glibc",
			},
		},
		test.GetParams())
}

//
func TestE2EDockerTagSetsPrune(t *testing.T) {
	tryConfig(test.NewTestHelper(t), "e2e/tagsets/docker-prune.yaml",
		0, 0, true, map[string][]string{
			"tagsets-docker/prune/busybox": {
				"1.31.1-musl", "1.31.1-glibc",
			},
		},
		test.GetParams())
}

//
func TestE2EDockerTagSetsRegex(t *testing.T) {
	tryConfig(test.NewTestHelper(t), "e2e/tagsets/docker-regex.yaml",
		0, 0, true, map[string][]string{
			"tagsets-docker/regex/busybox": {
				"1.26.1-musl", "1.26.1-glibc", "1.26.1-uclibc",
			},
			"tagsets-docker/regexinv/busybox": {"1.26.1-uclibc"},
		},
		test.GetParams())
}

//
func TestE2EDockerTagSetsLimit(t *testing.T) {
	tryConfig(test.NewTestHelper(t), "e2e/tagsets/docker-limit.yaml",
		0, 0, true, map[string][]string{
			"tagsets-docker/limit/busybox": {
				"1.36", "1.36.0", "1.36.0-glibc", "1.36.0-musl",
				"1.36.0-uclibc", "glibc",
			},
		},
		test.GetParams())
}

//
func TestE2EDockerTagSetsDigest(t *testing.T) {

	// NOTE: Docker does not allow push by digest reference, we therefore
	//       auto-generate a tag of the form `dregsy-{digest hex}`

	th := test.NewTestHelper(t)
	conf := tryConfig(th, "e2e/tagsets/docker-digest.yaml",
		0, 0, true, map[string][]string{
			"tagsets-docker/digest/busybox": {
				"dregsy-1d8a02c7a89283870e8dd6bb93dc66bc258e294491a6bbeb193a044ed88773ea",
				"1.35.0-uclibc",
			},
		},
		test.GetParams())

	validateDigests(th, conf, map[string][]string{
		"/tagsets-docker/digest/busybox": {
			"sha256:1d8a02c7a89283870e8dd6bb93dc66bc258e294491a6bbeb193a044ed88773ea",
			"sha256:ff4a7f382ff23a8f716741b6e60ef70a4986af3aff22d26e1f0e0cb4fde29289",
		},
	})
}

//
func TestE2ESkopeo(t *testing.T) {
	tryConfig(test.NewTestHelper(t), "e2e/base/skopeo.yaml",
		1, 0, true, nil, test.GetParams())
}

//
func TestE2ESkopeoPlatform(t *testing.T) {
	tryConfig(test.NewTestHelper(t), "e2e/base/skopeo-platform.yaml",
		1, 0, true, nil, test.GetParams())
}

//
func TestE2ESkopeoAllPlatforms(t *testing.T) {
	tryConfig(test.NewTestHelper(t), "e2e/base/skopeo-platform-all.yaml",
		1, 0, true, nil, test.GetParams())
}

//
func TestE2ESkopeoECR(t *testing.T) {
	registries.SkipIfECRNotConfigured(t, false)
	p := test.GetParams()
	registries.RemoveECRRepo(t, p, false)
	tryConfig(test.NewTestHelper(t), "e2e/base/skopeo-ecr.yaml",
		1, 0, true, nil, p)
	registries.RemoveECRRepo(t, p, false)
}

//
func TestE2ESkopeoECRPub(t *testing.T) {
	registries.SkipIfECRNotConfigured(t, true)
	p := test.GetParams()
	registries.RemoveECRRepo(t, p, true)
	tryConfig(test.NewTestHelper(t), "e2e/base/skopeo-ecr-pub.yaml",
		1, 0, true, nil, p)
	registries.RemoveECRRepo(t, p, true)
}

//
func TestE2ESkopeoECRPubNoAuth(t *testing.T) {
	tryConfig(test.NewTestHelper(t), "e2e/base/skopeo-ecr-pub-noauth.yaml",
		1, 0, true, nil, test.GetParams())
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
	registries.SkipIfECRNotConfigured(t, false)
	tryConfig(test.NewTestHelper(t), "e2e/mapping/skopeo-ecr.yaml",
		0, 0, true, map[string][]string{
			"mapping-skopeo/ecr/kubika/brucket":       {"v0.0.1"},
			"mapping-skopeo/ecr/kubika/brucket-shell": {"v0.0.1"},
		},
		test.GetParams())
}

//
func TestE2ESkopeoTagSetsRange(t *testing.T) {
	tryConfig(test.NewTestHelper(t), "e2e/tagsets/skopeo-range.yaml",
		0, 0, true, map[string][]string{
			"tagsets-skopeo/range/busybox": {
				"latest", "1.31", "1.31.0", "1.31.1-musl", "1.31.1",
				"1.31.1-uclibc", "1.31.1-glibc",
			},
		},
		test.GetParams())
}

//
func TestE2ESkopeoTagSetsPrune(t *testing.T) {
	tryConfig(test.NewTestHelper(t), "e2e/tagsets/skopeo-prune.yaml",
		0, 0, true, map[string][]string{
			"tagsets-skopeo/prune/busybox": {
				"1.31.1-musl", "1.31.1-glibc",
			},
		},
		test.GetParams())
}

//
func TestE2ESkopeoTagSetsRegex(t *testing.T) {
	tryConfig(test.NewTestHelper(t), "e2e/tagsets/skopeo-regex.yaml",
		0, 0, true, map[string][]string{
			"tagsets-skopeo/regex/busybox": {
				"1.26.1-musl", "1.26.1-glibc", "1.26.1-uclibc",
			},
			"tagsets-skopeo/regexinv/busybox": {"1.26.1-uclibc"},
		},
		test.GetParams())
}

//
func TestE2ESkopeoTagSetsLimit(t *testing.T) {
	tryConfig(test.NewTestHelper(t), "e2e/tagsets/skopeo-limit.yaml",
		0, 0, true, map[string][]string{
			"tagsets-skopeo/limit/busybox": {
				"1.36", "1.36.0", "1.36.0-glibc", "1.36.0-musl",
				"1.36.0-uclibc", "glibc",
			},
		},
		test.GetParams())
}

//
func TestE2ESkopeoTagSetsDigest(t *testing.T) {

	th := test.NewTestHelper(t)
	conf := tryConfig(th, "e2e/tagsets/skopeo-digest.yaml",
		0, 0, true, map[string][]string{
			"tagsets-skopeo/digest/busybox": {
				"1.35.0-uclibc",
			},
		},
		test.GetParams())

	validateDigests(th, conf, map[string][]string{
		"/tagsets-skopeo/digest/busybox": {
			"sha256:1d8a02c7a89283870e8dd6bb93dc66bc258e294491a6bbeb193a044ed88773ea",
			"sha256:ff4a7f382ff23a8f716741b6e60ef70a4986af3aff22d26e1f0e0cb4fde29289",
		},
	})
}

//
func tryConfig(th *test.TestHelper, file string, ticks int, wait time.Duration,
	verify bool, expectations map[string][]string, data interface{}) *sync.SyncConfig {

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
		return nil
	}

	log.Info("TEST - validating result")
	c, err := sync.LoadConfig(dst)
	th.AssertNoError(err)

	if expectations != nil {
		validateAgainstExpectations(th, c, expectations)
	} else {
		validateAgainstTaskMapping(th, c)
	}

	return c
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
			util.DecodeJSONAuth(t.Target.GetAuth()),
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
				util.DecodeJSONAuth(t.Target.GetAuth()),
				"", t.Target.SkipTLSVerify)
			th.AssertNoError(err)
			th.AssertEquivalentSlices(m.Tags, tags)
			validatePlatforms(th, ref, t, m)
		}
	}
}

//
func validatePlatforms(th *test.TestHelper, ref string, task *sync.Task,
	mapping *sync.Mapping) {

	if mapping.Platform == "" {
		return
	}

	plts := make(map[string]bool)

	if mapping.Platform == "all" {
		for _, p := range testPlatforms {
			plts[p] = true
		}
	} else {
		for _, p := range testPlatforms {
			plts[p] = p == mapping.Platform
		}
		plts[mapping.Platform] = true
	}

	for _, t := range mapping.Tags {

		for plt, exp := range plts {

			info, err := skopeo.Inspect(
				fmt.Sprintf("%s:%s", ref, t), plt, "{{.Os}}/{{.Architecture}}",
				util.DecodeJSONAuth(task.Target.GetAuth()),
				"", task.Target.SkipTLSVerify)
			th.AssertNoError(err)

			// FIXME: Skopeo inspect only shows OS and architecture, but not
			//        variant. Also, for platforms that are not present, it
			//        does not raise an error and instead returns info for the
			//        "default" platform. When testing syncing of a single
			//        platform, that's the default.
			var os, arch string
			if exp {
				os, arch, _ = util.SplitPlatform(plt)
			} else {
				os, arch, _ = util.SplitPlatform(mapping.Platform)
			}
			th.AssertEqual(fmt.Sprintf("%s/%s", os, arch), info)
		}
	}
}

//
func validateDigests(th *test.TestHelper, c *sync.SyncConfig,
	expectations map[string][]string) {

	for _, t := range c.Tasks {
		th.AssertNoError(t.Target.RefreshAuth())

		for _, m := range t.Mappings {
			ref := fmt.Sprintf("%s%s", t.Target.Registry, m.To)

			for _, d := range expectations[m.To] {
				info, err := skopeo.Inspect(
					fmt.Sprintf("%s@%s", ref, d), "", "{{.Digest}}",
					util.DecodeJSONAuth(t.Target.GetAuth()), "",
					t.Target.SkipTLSVerify)
				th.AssertNoError(err)
				th.AssertEqual(d, info)
			}
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
