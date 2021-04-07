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
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	log "github.com/sirupsen/logrus"

	"github.com/xelalexv/dregsy/internal/pkg/auth"
)

//
type DHRepoList struct {
	Items    []DHRepoDescriptor `json:"results",required`
	NextPage string             `json:"next",omitempty`
}

//
type DHRepoDescriptor struct {
	User        string `json:"user",required`
	Name        string `json:"name",required`
	Namespace   string `json:"namespace",required`
	Type        string `json:"repository_type",required`
	Description string `json:"description",omitempty`
	IsPrivate   bool   `json:"is_private",omitempty`

	// additional fields; include later on if needed
	//
	// status 				int
	// is_automated 		bool
	// can_edit 			bool
	// star_count 			int
	// pull_count 			int
	// last_updated 		time.Time e.g. "2020-04-27T14:25:06.739261Z"
	// is_migrated 			bool
	// collaborator_count	int
	// affiliation 			string
}

//
func newDockerhub(creds *auth.Credentials) ListSource {
	return &dockerhub{creds: creds}
}

//
type dockerhub struct {
	creds *auth.Credentials
}

//
func (d *dockerhub) Retrieve(maxItems int) ([]string, error) {

	var err error
	token := d.creds.Token()

	if token == nil || token.IsExpired() {
		token, err = d.getToken()
		if err != nil {
			return nil, err
		}
		d.creds.SetToken(token)
	} else {
		log.Debug("token already present and still valid")
	}

	var ret []string

	url := fmt.Sprintf(
		"https://hub.docker.com/v2/repositories/%s/?page_size=100",
		d.creds.Username())

	for {
		req, err := http.NewRequest("GET", url, nil)
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Authorization", fmt.Sprintf("JWT %s", token.Raw()))

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		var list DHRepoList
		if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
			return nil, err
		}

		for _, r := range list.Items {
			if r.Type == "image" {
				ret = append(ret, fmt.Sprintf("%s/%s", r.Namespace, r.Name))
			}
		}

		if list.NextPage == "" || (maxItems > 0 && len(ret) > maxItems) {
			return ret, nil
		}

		url = list.NextPage
	}
}

//
func (d *dockerhub) Ping() error {
	_, err := d.getToken()
	return err
}

//
func (d *dockerhub) getToken() (*auth.Token, error) {

	log.Debug("getting token")

	vals := url.Values{
		"username": {d.creds.Username()},
		"password": {d.creds.Password()},
	}

	resp, err := http.PostForm("https://hub.docker.com/v2/users/login/", vals)
	if err != nil {
		return nil, err
	}

	var res map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&res)
	if token, ok := res["token"].(string); ok {
		log.Debug("received token")
		return auth.NewToken(token), nil
	}
	return nil, fmt.Errorf("received token is not a string")
}
