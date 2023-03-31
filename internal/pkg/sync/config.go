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
	"path/filepath"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/fsnotify/fsnotify"
	log "github.com/sirupsen/logrus"

	"github.com/xelalexv/dregsy/internal/pkg/relays"
	"github.com/xelalexv/dregsy/internal/pkg/relays/docker"
	"github.com/xelalexv/dregsy/internal/pkg/relays/skopeo"
	"github.com/xelalexv/dregsy/internal/pkg/util"
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
	Watch      *bool               `yaml:"watch,omitempty"`
	//
	source string
	sha1   []byte
}

//
func (c *SyncConfig) ValidateSupport(s relays.Support) error {

	for _, t := range c.Tasks {
		for _, m := range t.Mappings {
			if err := s.Platform(m.Platform); err != nil {
				return err
			}
		}
	}

	return nil
}

//
func (c *SyncConfig) validate() error {

	if c.Watch == nil {
		log.Info(`

Note: Automatic restart after config file change is currently off by default. You can activate
      this by adding 'watch: true' to your config. The default will change to on in the future.

`)
	}

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
func (c *SyncConfig) watch() (*fsnotify.Watcher, error) {

	watch, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	if c.Watch == nil || !*c.Watch {
		log.Info("not watching config file")
		return watch, nil
	}

	// resolve any links
	if c.source, err = filepath.EvalSymlinks(c.source); err != nil {
		return nil, err
	}

	// make absolute
	if c.source, err = filepath.Abs(c.source); err != nil {
		return nil, err
	}

	if err = watch.Add(c.source); err != nil { // watch config file
		return nil, err
	}

	// In addition to the config file itself, we also watch the parent dir.
	// This is more robust. The file may be changed by replacing it, rather
	// than writing to it, which cannot be handled by the watch on the file.
	if err = watch.Add(filepath.Dir(c.source)); err != nil {
		return nil, err
	}

	// compute starting SHA1 digest of config file for later comparisons
	if c.sha1, err = util.ComputeSHA1(c.source); err != nil {
		return nil, err
	}

	log.WithField("file", c.source).Info(
		"watching config file, restarting on change")
	return watch, nil
}

//
func (c *SyncConfig) isChanged(evt fsnotify.Event) bool {

	log.WithFields(
		log.Fields{"op": evt.Op, "name": evt.Name}).Trace("file watch event")

	// event neither concerns the config file, nor its parent
	if evt.Name != c.source && evt.Name != filepath.Dir(c.source) {
		return false
	}

	log.WithField("op", evt.Op).Debug("config file event")

	// Removal of the parent dir is an indication for change: on Kubernetes,
	// config maps mounted into pods are updated by creating a new parent dir
	// and mounting new config map content into it. If the config file itself
	// was removed, we also see that as an indication for content change.
	if evt.Has(fsnotify.Remove) {
		if evt.Name == c.source {
			log.Debug("config file removed, assuming change")
		} else {
			log.Debug("config file parent directory removed, assuming change")
		}

	} else if evt.Has(fsnotify.Chmod) {
		// In case of a CHMOD event for the config file itself, we calculate
		// the SHA1 digest and compare with initial one to check for change.
		if evt.Name == c.source {
			d, err := util.ComputeSHA1(c.source)
			if err != nil || util.CompareSHA1(c.sha1, d) {
				log.Debug("no content change")
				return false
			}
			log.Debug("changed content")
		} else {
			return false // CHMOD on parent not relevant
		}

	} else {
		log.Debug("config file changed") // all other events mean change
	}

	return true
}

//
func LoadConfig(file string) (*SyncConfig, error) {

	data, err := ioutil.ReadFile(file)

	if err != nil {
		return nil, fmt.Errorf("error loading config file '%s': %v", file, err)
	}

	config := &SyncConfig{source: file}

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
