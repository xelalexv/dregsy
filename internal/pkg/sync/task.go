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
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"
	log "github.com/sirupsen/logrus"

	"github.com/xelalexv/dregsy/internal/pkg/registry"
	"github.com/xelalexv/dregsy/internal/pkg/relays/docker"
)

//
type Task struct {
	Name     string     `yaml:"name"`
	Interval int        `yaml:"interval"`
	Source   *Location  `yaml:"source"`
	Target   *Location  `yaml:"target"`
	Mappings []*Mapping `yaml:"mappings"`
	Verbose  bool       `yaml:"verbose"`
	//
	repoList *registry.RepoList
	ticker   *time.Ticker
	lastTick time.Time
	failed   bool
	//
	exit chan bool
	done chan bool
}

//
func (t *Task) validate() error {

	if len(t.Name) == 0 {
		return errors.New("a task requires a name")
	}

	if 0 < t.Interval && t.Interval < minimumTaskInterval {
		return fmt.Errorf(
			"minimum task interval is %d seconds", minimumTaskInterval)
	}

	if t.Interval < 0 {
		return errors.New("task interval needs to be 0 or a positive integer")
	}

	if err := t.Source.validate(); err != nil {
		return fmt.Errorf(
			"source registry in task '%s' invalid: %v", t.Name, err)
	}

	if err := t.Target.validate(); err != nil {
		return fmt.Errorf(
			"target registry in task '%s' invalid: %v", t.Name, err)
	}

	hasRegexp := false
	for _, m := range t.Mappings {
		if err := m.validate(); err != nil {
			return err
		}
		hasRegexp = hasRegexp || m.isRegexpFrom()
	}

	if hasRegexp {
		var err error
		s := t.Source
		if t.repoList, err = registry.NewRepoList(s.Registry, s.SkipTLSVerify,
			s.ListerType, s.ListerConfig, s.creds); err != nil {
			return fmt.Errorf(
				"cannot create repo list for task '%s': %v", t.Name, err)
		}
	}

	return nil
}

//
func (t *Task) startTicking(c chan *Task) {

	logger := log.WithField("task", t.Name)
	logger.Debug("task starts ticking")

	i := time.Duration(t.Interval)
	if i == 0 {
		i = 3
	}

	t.ticker = time.NewTicker(time.Second * i)
	t.lastTick = time.Now().Add(time.Second * i * (-2))

	t.exit = make(chan bool, 1)
	t.done = make(chan bool, 1)

	go func() {

		logger.Debug("sending initial fire")
		c <- t

		for {
			select {
			case <-t.ticker.C:
				logger.Debug("task firing")
				c <- t
			case <-t.exit:
				logger.Debug("task exiting")
				close(t.done)
				return
			}
		}
	}()
}

//
func (t *Task) tooSoon() bool {
	i := time.Duration(t.Interval)
	if i == 0 {
		return false
	}
	return time.Now().Before(t.lastTick.Add(time.Second * i / 2))
}

//
func (t *Task) stopTicking() {
	if t.ticker != nil {
		t.ticker.Stop()
		close(t.exit)
		<-t.done
	}
	log.WithField("task", t.Name).Debug("task exited")
}

//
func (t *Task) fail(f bool) {
	t.failed = t.failed || f
}

//
func (t *Task) mappingRefs(m *Mapping) ([][2]string, error) {

	var ret [][2]string

	if m != nil {

		if m.isRegexpFrom() {

			repos, err := t.repoList.Get()
			if err != nil {
				return nil, err
			}

			for _, r := range m.filterRepos(repos) {
				ret = append(ret, [2]string{
					t.Source.Registry + r,
					t.Target.Registry + m.mapPath(r),
				})
			}

		} else {
			ret = append(ret, [2]string{
				t.Source.Registry + m.From,
				t.Target.Registry + m.mapPath(m.From),
			})
		}
	}

	return ret, nil
}

//
func (t *Task) ensureTargetExists(ref string) error {

	isEcr, region, account := t.Target.GetECR()

	if isEcr {

		_, path, _ := docker.SplitRef(ref)
		if len(path) == 0 {
			return nil
		}

		sess, err := session.NewSession()
		if err != nil {
			return err
		}

		svc := ecr.New(sess, &aws.Config{
			Region: aws.String(region),
		})

		inpDescr := &ecr.DescribeRepositoriesInput{
			RegistryId:      aws.String(account),
			RepositoryNames: []*string{aws.String(path)},
		}

		out, err := svc.DescribeRepositories(inpDescr)
		if err == nil && len(out.Repositories) > 0 {
			log.WithField("ref", ref).Info("target already exists")
			return nil
		}

		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				if aerr.Code() != ecr.ErrCodeRepositoryNotFoundException {
					return err
				}
			} else {
				return err
			}
		}

		log.WithField("ref", ref).Info("creating target")
		inpCrea := &ecr.CreateRepositoryInput{
			RepositoryName: aws.String(path),
		}

		if _, err := svc.CreateRepository(inpCrea); err != nil {
			return err
		}
	}

	return nil
}
