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

	log "github.com/sirupsen/logrus"
)

//
type authRefresher interface {
	refresh() error
}

//
type Location struct {
	Registry      string         `yaml:"registry"`
	Auth          string         `yaml:"auth"`
	SkipTLSVerify bool           `yaml:"skip-tls-verify"`
	AuthRefresh   *time.Duration `yaml:"auth-refresh"`
	//
	refresher authRefresher
}

//
func (l *Location) validate() error {

	if l == nil {
		return errors.New("location is nil")
	}

	if l.Registry == "" {
		return errors.New("registry not set")
	}

	var interval time.Duration

	if l.AuthRefresh != nil {
		interval = *l.AuthRefresh
		if interval < minimumAuthRefreshInterval {
			interval = time.Duration(minimumAuthRefreshInterval)
			log.WithField("registry", l.Registry).
				Warnf("auth-refresh too short, setting to minimum: %s",
					minimumAuthRefreshInterval)
		}
	}

	if l.IsECR() {
		l.refresher = newECRAuthRefresher(l, interval)
	} else if interval > 0 {
		return fmt.Errorf(
			"'%s' wants authentication refresh, but is not an ECR registry",
			l.Registry)
	}

	if l.IsGCR() && l.Auth != "none" {
		l.refresher = newGCRAuthRefresher(l)
	}

	if l.Auth == "none" {
		l.Auth = ""
	}

	return nil
}

//
func (l *Location) RefreshAuth() error {
	if l.refresher == nil {
		return nil
	}
	return l.refresher.refresh()
}
