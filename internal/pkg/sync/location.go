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
	} else {
		l.creds = &auth.Credentials{}
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
		_, region, account := l.GetECR()
		l.creds.SetRefresher(auth.NewECRAuthRefresher(account, region, interval))
	} else if interval > 0 {
		return fmt.Errorf(
			"'%s' wants authentication refresh, but is not an ECR registry",
			l.Registry)
	}

	if l.IsGCR() && !disableAuth {
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
func (l *Location) IsECR() bool {
	ecr, _, _ := l.GetECR()
	return ecr
}

//
func (l *Location) GetECR() (ecr bool, region, account string) {
	return registry.IsECR(l.Registry)
}

//
func (l *Location) IsGCR() bool {
	return strings.HasSuffix(l.Registry, ".gcr.io")
}
