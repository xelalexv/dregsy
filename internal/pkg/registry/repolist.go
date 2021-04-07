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

package registry

import (
	"fmt"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/xelalexv/dregsy/internal/pkg/auth"
)

//
const defaultListerMaxItems = 100
const defaultListerCacheDuration = time.Hour

//
type ListSourceType string

const (
	Catalog   ListSourceType = "catalog"
	DockerHub                = "dockerhub"
	Index                    = "index"
)

//
func (t ListSourceType) IsValid() bool {
	switch t {
	case Catalog, DockerHub, Index:
		return true
	}
	return false
}

//
type ListSource interface {
	Ping() error
	Retrieve(maxItems int) ([]string, error)
}

//
func NewRepoList(registry string, insecure bool, typ ListSourceType,
	config map[string]string, creds *auth.Credentials) (*RepoList, error) {

	list := &RepoList{registry: registry}
	server := strings.SplitN(registry, ":", 2)[0]

	// DockerHub does not expose the registry catalog API, but separate APIs for
	// listing and searching. These APIs use tokens that are different from the
	// one used for normal registry actions, so we clone the credentials for list
	// use. For listing via catalog API, we can use the same credentials as for
	// push & pull.
	listCreds := creds
	if server == "registry.hub.docker.com" {
		var err error
		listCreds, err = auth.NewCredentialsFromBasic(
			creds.Username(), creds.Password())
		if err != nil {
			return nil, err
		}
		if typ != DockerHub && typ != Index {
			return nil, fmt.Errorf(
				"DockerHub only supports list types '%s' and '%s'",
				DockerHub, Index)
		}
	}

	switch typ {

	case DockerHub:
		list.source = newDockerhub(listCreds)

	case Index:
		if filter, ok := config["search"]; ok && filter != "" {
			list.source = newIndex(registry, filter, insecure, listCreds)
		} else {
			return nil, fmt.Errorf("index lister requires a search expression")
		}

	case Catalog, "":
		isECR, region, account := IsECR(registry)
		if isECR {
			// catalog can be used with ECR, but pagination doesn't work; it
			// requires an extra `NextToken` parameter which is not standard
			// and therefore not supported by the go-containerregistry remote
			// lib; if the registry is ECR we therefore use a dedicated ECR
			// lister based on the AWS Go SDK
			log.Info("using dedicated ECR lister instead of standard catalog")
			list.source = newECR(registry, region, account)
		} else {
			list.source = newCatalog(registry, insecure,
				strings.HasSuffix(server, ".gcr.io"), listCreds)
		}

	default:
		return nil, fmt.Errorf("invalid list source type '%s'", typ)
	}

	list.SetMaxItems(defaultListerMaxItems)
	list.SetCacheDuration(defaultListerCacheDuration)

	return list, nil
}

//
type RepoList struct {
	registry      string
	source        ListSource
	maxItems      int
	cacheDuration time.Duration
	expiry        time.Time
	repos         []string
}

//
func (l *RepoList) SetMaxItems(max int) {
	l.maxItems = max
}

//
func (l *RepoList) SetCacheDuration(d time.Duration) {
	l.cacheDuration = d
	l.expiry = time.Now()
	l.repos = nil
}

//
func (l *RepoList) isCacheValid() bool {
	return time.Now().Before(l.expiry)
}

//
func (l *RepoList) cacheList(repos []string) {
	if l.cacheDuration > 0 {
		log.Debug("caching repository list")
		l.expiry = time.Now().Add(l.cacheDuration)
		l.repos = repos
	} else {
		log.Debug("not caching repository list")
	}
}

//
func (l *RepoList) Get() ([]string, error) {

	if l.isCacheValid() {
		log.Debug("repository list still valid, re-using")
		return l.repos, nil
	}

	l.repos = nil
	log.Debug("retrieving repository list")

	if ret, err := l.source.Retrieve(l.maxItems); err != nil {
		return nil, err
	} else {
		l.cacheList(ret)
		return ret, nil
	}
}
