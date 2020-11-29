/*
 *
 */

package sync

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"io/ioutil"

	"gopkg.in/yaml.v2"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"

	"github.com/xelalexv/dregsy/internal/pkg/log"
	"github.com/xelalexv/dregsy/internal/pkg/relays/docker"
	"github.com/xelalexv/dregsy/internal/pkg/relays/skopeo"
)

//
const minimumTaskInterval = 30
const minimumAuthRefreshInterval = time.Hour

/* ----------------------------------------------------------------------------
 *
 */
type syncConfig struct {
	Relay      string              `yaml:"relay"`
	Docker     *docker.RelayConfig `yaml:"docker"`
	Skopeo     *skopeo.RelayConfig `yaml:"skopeo"`
	DockerHost string              `yaml:"dockerhost"`  // DEPRECATED
	APIVersion string              `yaml:"api-version"` // DEPRECATED
	Tasks      []*task             `yaml:"tasks"`
}

//
func (c *syncConfig) validate() error {

	if c.Relay == "" {
		c.Relay = docker.RelayID
	}

	switch c.Relay {

	case docker.RelayID:
		if c.Docker == nil {
			if c.DockerHost == "" && c.APIVersion == "" {
				log.Warning(
					"not specifying the 'docker' config item is deprecated")
			}
			templ := "the top-level '%s' setting is deprecated, " +
				"use 'docker' config item instead"
			if c.DockerHost != "" {
				log.Warning(fmt.Sprintf(templ, "dockerhost"))
			}
			if c.APIVersion != "" {
				log.Warning(fmt.Sprintf(templ, "api-version"))
			}
			c.Docker = &docker.RelayConfig{
				DockerHost: c.DockerHost,
				APIVersion: c.APIVersion,
			}

		} else {
			templ := "discarding deprecated top-level '%s' setting and " +
				"using 'docker' config item instead"
			if c.DockerHost != "" {
				log.Warning(fmt.Sprintf(templ, "dockerhost"))
				c.DockerHost = ""
			}
			if c.APIVersion != "" {
				log.Warning(fmt.Sprintf(templ, "api-version"))
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
type task struct {
	Name     string     `yaml:"name"`
	Interval int        `yaml:"interval"`
	Source   *location  `yaml:"source"`
	Target   *location  `yaml:"target"`
	Mappings []*mapping `yaml:"mappings"`
	Verbose  bool       `yaml:"verbose"`
	//
	ticker   *time.Ticker
	lastTick time.Time
	failed   bool
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
func (t *task) tooSoon() bool {
	i := time.Duration(t.Interval)
	if i == 0 {
		return false
	}
	return time.Now().Before(t.lastTick.Add(time.Second * i / 2))
}

//
func (t *task) stopTicking(c chan *task) {
	if t.ticker != nil {
		t.ticker.Stop()
		t.ticker = nil
	}
}

//
func (t *task) fail(f bool) {
	t.failed = t.failed || f
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
			log.Info("target '%s' already exists", ref)
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

		log.Info("creating target '%s'", ref)
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
	Registry      string         `yaml:"registry"`
	Auth          string         `yaml:"auth"`
	SkipTLSVerify bool           `yaml:"skip-tls-verify"`
	AuthRefresh   *time.Duration `yaml:"auth-refresh"`
	lastRefresh   time.Time
	expiry        time.Time
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
			log.Warning(
				"auth-refresh for '%s' too short, setting to minimum: %s",
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
func (l *location) refreshAuth() error {
	if l.isGCR() {
		return l.refreshAuthGCP()
	}

	if l.AuthRefresh == nil || time.Since(l.lastRefresh) < *l.AuthRefresh {
		return nil
	}

	_, region, account := l.getECR()
	log.Info("refreshing credentials for '%s'", l.Registry)

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

//
func (l *location) refreshAuthGCP() error {
	if l.expiry.Sub(time.Now()) > 0 {
		return nil
	}

	var runner func() (string, time.Time, error)

	if os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") != "" {
		runner = tokenFromCreds
	} else if isGCE() {
		runner = tokenFromMetadata
	} else {
		return fmt.Errorf("No GOOGLE_APPLICATION_CREDENTIALS set, or not a GCE instance")
	}

	authToken, expiry, err := runner()
	log.Info("refreshing credentials for '%s'", l.Registry)

	if err != nil || authToken == "" {
		return err
	}

	user := "oauth2accesstoken"
	pass := authToken

	l.Auth = base64.StdEncoding.EncodeToString([]byte(
		fmt.Sprintf("{\"username\": \"%s\", \"password\": \"%s\"}",
			user, pass)))

	l.expiry = expiry

	return nil
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
