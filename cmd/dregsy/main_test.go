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

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"

	"github.com/xelalexv/dregsy/internal/pkg/relays/skopeo"
	"github.com/xelalexv/dregsy/internal/pkg/sync"
	"github.com/xelalexv/dregsy/internal/pkg/test"
)

//
const (
	EnvAccessKeyID     = "AWS_ACCESS_KEY_ID"
	EnvSecretAccessKey = "AWS_SECRET_ACCESS_KEY"
	EnvECRRegistry     = "DREGSY_TEST_ECR_REGISTRY"
	EnvECRRepo         = "DREGSY_TEST_ECR_REPO"
)

//
type TestParams struct {
	ECRRegistry string
	ECRRepo     string
}

//
func getTestParams() *TestParams {
	ret := &TestParams{
		ECRRegistry: os.Getenv(EnvECRRegistry),
		ECRRepo:     os.Getenv(EnvECRRepo),
	}
	if ret.ECRRepo == "" {
		ret.ECRRepo = "dregsy/test"
	}
	return ret
}

//
func TestE2EDocker(t *testing.T) {
	tryConfig(test.NewTestHelper(t), "e2e/docker.yaml", true, nil)
}

//
func TestE2EDockerECR(t *testing.T) {
	skipIfECRNotConfigured(t)
	p := getTestParams()
	removeECRRepo(p.ECRRegistry, p.ECRRepo)
	tryConfig(test.NewTestHelper(t), "e2e/docker-ecr.yaml", true, p)
	removeECRRepo(p.ECRRegistry, p.ECRRepo)
}

//
func TestE2ESkopeo(t *testing.T) {
	tryConfig(test.NewTestHelper(t), "e2e/skopeo.yaml", true, nil)
}

//
func TestE2ESkopeoECR(t *testing.T) {
	skipIfECRNotConfigured(t)
	p := getTestParams()
	removeECRRepo(p.ECRRegistry, p.ECRRepo)
	tryConfig(test.NewTestHelper(t), "e2e/skopeo-ecr.yaml", true, p)
	removeECRRepo(p.ECRRegistry, p.ECRRepo)
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
		instance.Shutdown()
	}

	for i := 0; i < 10; i++ {
		select {
		case <-testSync:
			return dregsyExitCode
		default:
			time.Sleep(time.Second)
		}
	}

	panic("dregsy did not stop")
}

//
func skipIfECRNotConfigured(t *testing.T) {
	var missing []string
	if os.Getenv(EnvAccessKeyID) == "" {
		missing = append(missing, EnvAccessKeyID)
	}
	if os.Getenv(EnvSecretAccessKey) == "" {
		missing = append(missing, EnvSecretAccessKey)
	}
	if os.Getenv(EnvECRRegistry) == "" {
		missing = append(missing, EnvECRRegistry)
	}
	if len(missing) > 0 {
		t.Skipf("skipping, ECR not configured, missing these environment variables: %v",
			missing)
	}
}

//
func removeECRRepo(registry, repo string) error {

	loc := &sync.Location{Registry: registry}
	isEcr, region, _ := loc.GetECR()

	if !isEcr {
		return nil
	}

	sess, err := session.NewSession()
	if err != nil {
		return err
	}

	svc := ecr.New(sess, &aws.Config{
		Region: aws.String(region),
	})

	inpDel := &ecr.DeleteRepositoryInput{
		Force:          aws.Bool(true),
		RepositoryName: aws.String(repo),
	}

	_, err = svc.DeleteRepository(inpDel)

	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() == ecr.ErrCodeRepositoryNotFoundException {
				return nil
			}
		}
	}

	return err
}
