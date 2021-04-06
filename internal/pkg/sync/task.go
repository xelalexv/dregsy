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
	"io/ioutil"
	"strings"
	"time"

    "gopkg.in/yaml.v2"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"
	log "github.com/sirupsen/logrus"

	"github.com/xelalexv/dregsy/internal/pkg/relays/docker"
)

type MappingList struct {
    Mappings []*Mapping `mappings`
}

//
type Task struct {
	Name     string     `yaml:"name"`
	Interval int        `yaml:"interval"`
	Source   *Location  `yaml:"source"`
	Target   *Location  `yaml:"target"`
	MappingFile *string  `yaml:"mappings_file"`
	Mappings []*Mapping `yaml:"mappings"`
	Verbose  bool       `yaml:"verbose"`

	//
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


    if t.MappingFile != nil {
        err := t.refreshMapping()
        if err != nil {
            return fmt.Errorf("failed to procure mappings", t.Name, err)
        }
    } else{
        t.validateMappings()
    }

	return nil
}

func(t *Task) validateMappings() error {
    for _, m := range t.Mappings {
        if err := m.validate(); err != nil {

            return fmt.Errorf("error parsing mappings '%s': %v", m.From, err)
        }
        m.From = normalizePath(m.From)
        m.To = normalizePath(m.To)
    }
    return nil
}

func(t *Task) refreshMapping() error {

    logger := log.WithField("task", t.Name)

    mappingFile := *t.MappingFile

    data, err := ioutil.ReadFile(mappingFile)

    if err != nil{
        return fmt.Errorf("issue with mapping file  '%s': %v", mappingFile, err)
    }

    maplist := &MappingList{}

    if err = yaml.Unmarshal(data, maplist); err != nil {
        return fmt.Errorf("error parsing mappings config file '%s': %v", mappingFile, err)
    }

    t.Mappings = make([]*Mapping, len(maplist.Mappings))

    //t.Mappings = maplist.Mappings

    copy(t.Mappings, maplist.Mappings)

    logger.Info("refreshed task list from file '%s'", mappingFile)

    t.validateMappings()

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
func (t *Task) mappingRefs(m *Mapping) (from, to string) {
	if m != nil {
		from = t.Source.Registry + m.From
		to = t.Target.Registry + m.To
	}
	return from, to
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

//
func normalizePath(p string) string {
	if strings.HasPrefix(p, "/") {
		return p
	}
	return "/" + p
}
