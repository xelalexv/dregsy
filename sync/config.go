/*
 *
 */

package sync

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
	"io/ioutil"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"

	"github.com/xelalexv/dregsy/docker"
)

//
const minimumTaskInterval = 30
const minimumAuthRefreshInterval = time.Hour

/* ----------------------------------------------------------------------------
 *
 */
type syncConfig struct {
	DockerHost string  `yaml:"dockerhost"`
	APIVersion string  `yaml:"api-version"`
	Tasks      []*task `yaml:"tasks"`
}

//
func (c *syncConfig) validate() error {
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
type task struct {
	Name     string     `yaml:"name"`
	Interval int        `yaml:"interval"`
	Source   *location  `yaml:"source"`
	Target   *location  `yaml:"target"`
	Mappings []*mapping `yaml:"mappings"`
	Verbose  bool       `yaml:"verbose"`
	//
	ticker *time.Ticker
}

//
func (t *task) validate() error {

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
func (t *task) startTicking(c chan *task) {

	i := time.Duration(t.Interval)

	if i == 0 {
		i = 3
	}

	t.ticker = time.NewTicker(time.Second * i)

	go func() {
		// fire once right at the start
		c <- t
		for range t.ticker.C {
			c <- t
		}
	}()
}

//
func (t *task) stopTicking(c chan *task) {
	if t.ticker != nil {
		t.ticker.Stop()
		t.ticker = nil
	}
}

//
func (t *task) mappingRefs(m *mapping) (from, to string) {
	if m != nil {
		from = t.Source.Registry + m.From
		to = t.Target.Registry + m.To
	}
	return from, to
}

//
func (t *task) ensureTargetExists(ref string) error {

	isEcr, region, account := t.Target.getECR()

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
			LogInfo("target '%s' already exists", ref)
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

		LogInfo("creating target '%s'", ref)
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
type location struct {
	Registry    string         `yaml:"registry"`
	Auth        string         `yaml:"auth"`
	AuthRefresh *time.Duration `yaml:"auth-refresh"`
	lastRefresh time.Time
}

//
func (l *location) validate() error {

	if l == nil {
		return errors.New("location is nil")
	}

	if l.Registry == "" {
		return errors.New("registry not set")
	}

	l.lastRefresh = time.Time{}

	if l.AuthRefresh != nil {

		if *l.AuthRefresh == 0 {
			l.AuthRefresh = nil

		} else if !l.isECR() {
			return fmt.Errorf(
				"'%s' wants authentication refresh, but is not an ECR registry",
				l.Registry)

		} else if *l.AuthRefresh < minimumAuthRefreshInterval {
			*l.AuthRefresh = time.Duration(minimumAuthRefreshInterval)
			LogWarning("auth-refresh for '%s' too short, setting to minimum: %s",
				l.Registry, minimumAuthRefreshInterval)
		}
	}

	return nil
}

//
func (l *location) isECR() bool {
	ecr, _, _ := l.getECR()
	return ecr
}

//
func (l *location) getECR() (ecr bool, region, account string) {
	url := strings.Split(l.Registry, ".")
	ecr = len(url) == 6 && url[1] == "dkr" && url[2] == "ecr" &&
		url[4] == "amazonaws" && url[5] == "com"
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
func (l *location) refreshAuth() error {

	if l.AuthRefresh == nil || time.Since(l.lastRefresh) < *l.AuthRefresh {
		return nil
	}

	_, region, account := l.getECR()
	LogInfo("refreshing credentials for '%s'", l.Registry)

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
		l.lastRefresh = time.Now()

		return nil
	}

	return fmt.Errorf("no authorization data for")
}

/* ----------------------------------------------------------------------------
 *
 */
type mapping struct {
	From string   `yaml:"from"`
	To   string   `yaml:"to"`
	Tags []string `yaml:"tags"`
}

//
func (m *mapping) validate() error {

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
func LoadConfig(file string) (*syncConfig, error) {

	data, err := ioutil.ReadFile(file)

	if err != nil {
		return nil, fmt.Errorf("error loading config file '%s': %v", file, err)
	}

	config := &syncConfig{}
	err = yaml.Unmarshal(data, config)

	if err != nil {
		return nil, fmt.Errorf("error parsing config file '%s': %v", file, err)
	}

	return config, config.validate()
}
