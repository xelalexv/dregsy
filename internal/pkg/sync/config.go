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
	"fmt"
	"io/ioutil"
	"time"

	"gopkg.in/yaml.v2"

	log "github.com/sirupsen/logrus"

	"github.com/xelalexv/dregsy/internal/pkg/relays/docker"
	"github.com/xelalexv/dregsy/internal/pkg/relays/skopeo"
)

//
const minimumTaskInterval = 30
const minimumAuthRefreshInterval = time.Hour

//
type SyncConfig struct {
	Relay      string              `yaml:"relay"`
	Docker     *docker.RelayConfig `yaml:"docker"`
	Skopeo     *skopeo.RelayConfig `yaml:"skopeo"`
	DockerHost string              `yaml:"dockerhost"`  // DEPRECATED
	APIVersion string              `yaml:"api-version"` // DEPRECATED
	Lister     *ListerConfig       `yaml:"lister"`
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

	if err := c.Lister.validate(); err != nil {
		return err
	}

	for _, t := range c.Tasks {
		if err := t.validate(); err != nil {
			return err
		}
		if c.Lister != nil && t.repoList != nil {
			if c.Lister.MaxItems != 0 {
				t.repoList.SetMaxItems(c.Lister.MaxItems)
			}
			if c.Lister.CacheDuration != 0 {
				t.repoList.SetCacheDuration(c.Lister.CacheDuration)
			}
		}
	}

	return nil
}

//
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

//
type ListerConfig struct {
	MaxItems      int           `yaml:"maxItems"`
	CacheDuration time.Duration `yaml:"cacheDuration"`
}

//
func (c *ListerConfig) validate() error {

	if c == nil {
		return nil
	}

	if c.MaxItems < 0 {
		log.Warn(
			"lister items set to unlimited, may cause excessive network traffic")
	} else {
		log.Debugf("lister max items set to %d", c.MaxItems)
	}

	if c.CacheDuration < 0 {
		log.Warn("lister cache turned off, " +
			"may cause excessive network traffic and/or rate limits")
	}
	log.Debugf("lister cache duration set to %v", c.CacheDuration)

	return nil
}
