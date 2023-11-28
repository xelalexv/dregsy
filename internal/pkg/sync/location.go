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
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/xelalexv/dregsy/internal/pkg/auth"
	"github.com/xelalexv/dregsy/internal/pkg/registry"
)

//
type Location struct {
	Registry      string            `yaml:"registry"`
	Auth          string            `yaml:"auth"`
	SkipTLSVerify bool              `yaml:"skip-tls-verify"`
	AuthRefresh   *time.Duration    `yaml:"auth-refresh"`
	ListerConfig  map[string]string `yaml:"lister"`
	ListerType    registry.ListSourceType
	//
	ecr     bool
	public  bool
	region  string
	account string
	//
	creds *auth.Credentials
}

//
func (l *Location) validate() error {

	if l == nil {
		return errors.New("location is nil")
	}

	if l.Registry == "" {
		return errors.New("registry not set")
	}

	if l.ListerConfig != nil {
		if typ, ok := l.ListerConfig["type"]; ok {
			l.ListerType = registry.ListSourceType(typ)
			if !l.ListerType.IsValid() {
				return fmt.Errorf("invalid lister type: %s", l.ListerType)
			}
		} else {
			return fmt.Errorf("no lister type set")
		}
	}

	disableAuth := l.Auth == "none"
	if disableAuth {
		l.Auth = ""
	}

	// move Auth into credentials
	if l.Auth != "" {
		crd, err := auth.NewCredentialsFromAuth(l.Auth)
		if err != nil {
			return fmt.Errorf("invalid Auth: %v", err)
		}
		l.creds = crd
		l.Auth = ""

		log.WithFields(log.Fields{
			"registry": l.Registry,
			"username": l.creds.Username(),
		}).Info("using credentials from config")

	} else {
		l.creds = &auth.Credentials{}
	}

	l.ecr, l.public, l.region, l.account = registry.IsECR(l.Registry)

	if l.ecr && l.public {
		p := strings.Split(l.Registry, "@")
		if len(p) > 1 {
			l.Registry = p[1]
		}
	}

	var interval time.Duration

	if l.AuthRefresh != nil {
		interval = *l.AuthRefresh
		if interval < minimumAuthRefreshInterval {
			interval = time.Duration(minimumAuthRefreshInterval)
			log.WithFields(log.Fields{
				"registry": l.Registry,
				"minimum":  minimumAuthRefreshInterval,
			}).Warn("auth-refresh too short, setting to minimum")
		}
	}

	if l.ecr {
		l.creds.SetRefresher(
			auth.NewECRAuthRefresher(l.public, l.account, l.region, interval))
	} else if interval > 0 {
		return fmt.Errorf(
			"'%s' wants authentication refresh, but is not an ECR registry",
			l.Registry)
	}

	// If the credentials were provided we're assuming the user wants to use
	// them and not configure the refresher, otherwise (unless auth is disabled)
	// we'll use the GCR refresher.
	if l.IsGCR() && (!disableAuth || l.creds.Empty()) {
		l.creds.SetRefresher(auth.NewGCRAuthRefresher())
	}

	return nil
}

//
func (l *Location) GetAuth() string {
	if l.creds != nil {
		return l.creds.Auth()
	}
	log.WithField("registry", l.Registry).Debug("no credentials")
	return ""
}

//
func (l *Location) RefreshAuth() error {
	if l.creds == nil {
		return nil
	}
	log.WithField("registry", l.Registry).Info("refreshing credentials")
	return l.creds.Refresh()
}

//
func (l *Location) IsECR() (bool, bool) {
	return l.ecr, l.public
}

//
func (l *Location) GetECR() (bool, bool, string, string) {
	return l.ecr, l.public, l.region, l.account
}

//
func (l *Location) IsGCR() bool {
	return strings.HasSuffix(l.Registry, "gcr.io") ||
		strings.HasSuffix(l.Registry, "-docker.pkg.dev")
}
