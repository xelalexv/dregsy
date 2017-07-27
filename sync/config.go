/*
 *
 */

package sync

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
	"io/ioutil"
)

//
const minimumTaskInterval = 30

//
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

//
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
		return fmt.Errorf("minimum task interval is %d seconds", minimumTaskInterval)
	}
	if t.Interval < 0 {
		return errors.New("task interval needs to be 0 or a positive integer")
	}
	if err := t.Source.validate(); err != nil {
		return fmt.Errorf("source registry in task '%s' invalid: %v", t.Name, err)
	}
	if err := t.Target.validate(); err != nil {
		return fmt.Errorf("target registry in task '%s' invalid: %v", t.Name, err)
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
func normalizePath(p string) string {
	if strings.HasPrefix(p, "/") {
		return p
	}
	return "/" + p
}

//
type location struct {
	Registry string `yaml:"registry"`
	Auth     string `yaml:"auth"`
}

//
func (l *location) validate() error {
	if l == nil {
		return errors.New("location is nil")
	}
	if l.Registry == "" {
		return errors.New("registry not set")
	}
	return nil
}

//
type mapping struct {
	From string `yaml:"from"`
	To   string `yaml:"to"`
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

//
// load config from YAML file
//

//
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
