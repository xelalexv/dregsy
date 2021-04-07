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
	"context"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/registry"

	"github.com/xelalexv/dregsy/internal/pkg/auth"
)

//
func newIndex(reg, filter string, insecure bool, creds *auth.Credentials) ListSource {

	ret := &index{filter: filter}

	if !isDockerHub(reg) {
		ret.filter = fmt.Sprintf("%s/%s", reg, filter)
	}

	ret.auth = &types.AuthConfig{
		Username: creds.Username(),
		Password: creds.Password(),
	}
	if creds.Token() != nil {
		ret.auth.RegistryToken = creds.Token().Raw()
	}

	ret.opts = &registry.ServiceOptions{}
	if insecure {
		ret.opts.InsecureRegistries = []string{reg}
	}

	ret.auth.ServerAddress = reg

	return ret
}

//
type index struct {
	opts   *registry.ServiceOptions
	auth   *types.AuthConfig
	filter string
}

//
func (i *index) Retrieve(maxItems int) ([]string, error) {

	svc, err := registry.NewService(*i.opts)
	if err != nil {
		return nil, err
	}

	// FIXME: consider using token
	res, err := svc.Search(
		context.TODO(), i.filter, maxItems, i.auth, "dregsy", nil)
	if err != nil {
		return nil, err
	}

	ret := make([]string, 0, res.NumResults)
	for _, r := range res.Results {
		ret = append(ret, r.Name)
	}

	return ret, nil
}

//
func (i *index) Ping() error {
	svc, err := registry.NewService(*i.opts)
	if err != nil {
		return err
	}
	if _, _, err := svc.Auth(context.TODO(), i.auth, "dregsy"); err != nil {
		return err
	}
	return nil
}

//
func isDockerHub(reg string) bool {
	return reg == "" || reg == "docker.com" || reg == "docker.io" ||
		strings.HasSuffix(reg, ".docker.com") ||
		strings.HasSuffix(reg, ".docker.io")
}
