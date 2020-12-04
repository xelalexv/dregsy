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
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"
	log "github.com/sirupsen/logrus"

	"github.com/xelalexv/dregsy/internal/pkg/relays/docker"
	"github.com/xelalexv/dregsy/internal/pkg/relays/skopeo"
)

//
const minimumTaskInterval = 30
const minimumAuthRefreshInterval = time.Hour

/* ----------------------------------------------------------------------------
 *
 */
type SyncConfig struct {
	Relay      string              `yaml:"relay"`
	Docker     *docker.RelayConfig `yaml:"docker"`
	Skopeo     *skopeo.RelayConfig `yaml:"skopeo"`
	DockerHost string              `yaml:"dockerhost"`  // DEPRECATED
	APIVersion string              `yaml:"api-version"` // DEPRECATED
	Tasks      []*Task             `yaml:"tasks"`
}

//
func (c *SyncConfig) validate() error {

	if c.Relay == "" {
		c.Relay = docker.RelayID
	}

	switch c.Relay {

	case docker.RelayID:
		if c.Docker == nil {
			if c.DockerHost == "" && c.APIVersion == "" {
				log.Warn("not specifying the 'docker' config item is deprecated")
			}
			templ := "the top-level '%s' setting is deprecated, " +
				"use 'docker' config item instead"
			if c.DockerHost != "" {
				log.Warnf(templ, "dockerhost")
			}
			if c.APIVersion != "" {
				log.Warnf(templ, "api-version")
			}
			c.Docker = &docker.RelayConfig{
				DockerHost: c.DockerHost,
				APIVersion: c.APIVersion,
			}

		} else {
			templ := "discarding deprecated top-level '%s' setting and " +
				"using 'docker' config item instead"
			if c.DockerHost != "" {
				log.Warnf(templ, "dockerhost")
				c.DockerHost = ""
			}
			if c.APIVersion != "" {
				log.Warnf(templ, "api-version")
				c.APIVersion = ""
			}
		}

	case skopeo.RelayID:
		if c.DockerHost != "" {
			return fmt.Errorf(
				"setting 'dockerhost' implies '%s' relay, but relay is set to '%s'",
				docker.RelayID, c.Relay)
		}

	default:
		return fmt.Errorf(
			"invalid relay type: '%s', must be either '%s' or '%s'",
			c.Relay, docker.RelayID, skopeo.RelayID)
	}

	for _, t := range c.Tasks {
		if err := t.validate(); err != nil {
			return err
		}
	}
	return nil
}

/* ----------------------------------------------------------------------------
 *
 */
type Task struct {
	Name     string     `yaml:"name"`
	Interval int        `yaml:"interval"`
	Source   *Location  `yaml:"source"`
	Target   *Location  `yaml:"target"`
	Mappings []*Mapping `yaml:"mappings"`
	Verbose  bool       `yaml:"verbose"`
	//
	ticker   *time.Ticker
	lastTick time.Time
	failed   bool
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

	for _, m := range t.Mappings {
		if err := m.validate(); err != nil {
			return err
		}
		m.From = normalizePath(m.From)
		m.To = normalizePath(m.To)
	}

	return nil
}

//
func (t *Task) startTicking(c chan *Task) {

	i := time.Duration(t.Interval)

	if i == 0 {
		i = 3
	}

	t.ticker = time.NewTicker(time.Second * i)
	t.lastTick = time.Now().Add(time.Second * i * (-2))

	go func() {
		// fire once right at the start
		c <- t
		for range t.ticker.C {
			c <- t
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
func (t *Task) stopTicking(c chan *Task) {
	if t.ticker != nil {
		t.ticker.Stop()
		t.ticker = nil
	}
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

/* ----------------------------------------------------------------------------
 *
 */
type Location struct {
	Registry            string         `yaml:"registry"`
	Auth                string         `yaml:"auth"`
	SkipTLSVerify       bool           `yaml:"skip-tls-verify"`
	AuthRefresh         *time.Duration `yaml:"auth-refresh"`
	ecrTokenLastRefresh time.Time
	gcrTokenExpiry      time.Time
}

//
func (l *Location) validate() error {

	if l == nil {
		return errors.New("location is nil")
	}

	if l.Registry == "" {
		return errors.New("registry not set")
	}

	l.ecrTokenLastRefresh = time.Time{}

	if l.AuthRefresh != nil {

		if *l.AuthRefresh == 0 {
			l.AuthRefresh = nil

		} else if !l.IsECR() {
			return fmt.Errorf(
				"'%s' wants authentication refresh, but is not an ECR registry",
				l.Registry)

		} else if *l.AuthRefresh < minimumAuthRefreshInterval {
			*l.AuthRefresh = time.Duration(minimumAuthRefreshInterval)
			log.WithField("registry", l.Registry).
				Warnf("auth-refresh too short, setting to minimum: %s",
					minimumAuthRefreshInterval)
		}
	}

	return nil
}

//
func (l *Location) IsECR() bool {
	ecr, _, _ := l.GetECR()
	return ecr
}

//
func (l *Location) GetECR() (ecr bool, region, account string) {
	url := strings.Split(l.Registry, ".")
	ecr = (len(url) == 6 || len(url) == 7) && url[1] == "dkr" && url[2] == "ecr" &&
		url[4] == "amazonaws" && url[5] == "com" && (len(url) == 6 || url[6] == "cn")
	if ecr {
		region = url[3]
		account = url[0]
	} else {
		region = ""
		account = ""
	}
	return
}

//
func (l *Location) RefreshAuth() error {

	if l.isGCR() {
		return l.refreshAuthGCP()
	}

	if l.AuthRefresh == nil || time.Since(l.ecrTokenLastRefresh) < *l.AuthRefresh {
		return nil
	}

	_, region, account := l.GetECR()
	log.WithField("registry", l.Registry).Info("refreshing credentials")

	sess, err := session.NewSession()

	if err != nil {
		return err
	}

	svc := ecr.New(sess, &aws.Config{
		Region: aws.String(region),
	})

	input := &ecr.GetAuthorizationTokenInput{
		RegistryIds: []*string{aws.String(account)},
	}

	authToken, err := svc.GetAuthorizationToken(input)
	if err != nil {
		return err
	}

	for _, data := range authToken.AuthorizationData {

		output, err := base64.StdEncoding.DecodeString(*data.AuthorizationToken)
		if err != nil {
			return err
		}

		split := strings.Split(string(output), ":")
		if len(split) != 2 {
			return fmt.Errorf("failed to parse credentials")
		}

		user := strings.TrimSpace(split[0])
		pass := strings.TrimSpace(split[1])

		l.Auth = base64.StdEncoding.EncodeToString([]byte(
			fmt.Sprintf("{\"username\": \"%s\", \"password\": \"%s\"}",
				user, pass)))
		l.ecrTokenLastRefresh = time.Now()

		return nil
	}

	return fmt.Errorf("no authorization data for")
}

//
func (l *Location) refreshAuthGCP() error {

	if l.gcrTokenExpiry.Sub(time.Now()) > 0 {
		return nil
	}

	var (
		authToken string
		expiry    time.Time
		err       error
	)

	if os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") != "" {
		authToken, expiry, err = tokenFromCreds()
	} else if isGCE() {
		authToken, expiry, err = tokenFromMetadata()
	} else {
		return fmt.Errorf(
			"No GOOGLE_APPLICATION_CREDENTIALS set, or not a GCE instance")
	}

	log.WithField("registry", l.Registry).Info("refreshing credentials")

	if err != nil || authToken == "" {
		return err
	}

	l.Auth = base64.StdEncoding.EncodeToString([]byte(
		fmt.Sprintf(
			"{\"username\": \"oauth2accesstoken\", \"password\": \"%s\"}",
			authToken)))

	l.gcrTokenExpiry = expiry

	return nil
}

/* ----------------------------------------------------------------------------
 *
 */
type Mapping struct {
	From string   `yaml:"from"`
	To   string   `yaml:"to"`
	Tags []string `yaml:"tags"`
}

//
func (m *Mapping) validate() error {

	if m == nil {
		return errors.New("mapping is nil")
	}

	if m.From == "" {
		return errors.New("mapping without 'From' path")
	}

	if m.To == "" {
		m.To = m.From
	}

	return nil
}

/* ----------------------------------------------------------------------------
 * load config from YAML file
 */
func LoadConfig(file string) (*SyncConfig, error) {

	data, err := ioutil.ReadFile(file)

	if err != nil {
		return nil, fmt.Errorf("error loading config file '%s': %v", file, err)
	}

	config := &SyncConfig{}

	if err = yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("error parsing config file '%s': %v", file, err)
	}

	if err = config.validate(); err != nil {
		return nil, err
	}

	return config, nil
}
