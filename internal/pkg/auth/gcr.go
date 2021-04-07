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

package auth

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

//
const gcp_metadata_url = "http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/token"

//
func NewGCRAuthRefresher() Refresher {
	return &gcrAuthRefresher{}
}

//
type gcrAuthRefresher struct {
	expiry time.Time
}

//
func (rf *gcrAuthRefresher) Refresh(creds *Credentials) error {

	if time.Now().Before(rf.expiry) {
		return nil
	}

	var authToken string
	var expiry time.Time
	var err error

	if os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") != "" {
		authToken, expiry, err = gcpTokenFromCreds()

	} else if isGCEInstance() {
		authToken, expiry, err = gcpTokenFromMetadata()

	} else {
		return fmt.Errorf(
			"neither GOOGLE_APPLICATION_CREDENTIALS set, nor a GCE instance")
	}

	if err != nil {
		return err
	}
	if authToken == "" {
		return fmt.Errorf("no auth token received")
	}

	creds.username = "oauth2accesstoken"
	creds.password = authToken
	creds.auther = BasicAuthJSON
	rf.expiry = expiry

	return nil
}

//
func isGCEInstance() bool {
	resp, err := http.Head(gcp_metadata_url)
	if err != nil {
		return false
	}
	return resp.Header.Get("Metadata-Flavor") == "Google"
}

//
func gcpTokenFromCreds() (string, time.Time, error) {

	b, err := ioutil.ReadFile(os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"))
	if err != nil {
		return "", time.Time{}, err
	}

	conf, err := google.JWTConfigFromJSON(
		b, "https://www.googleapis.com/auth/devstorage.read_write")
	if err != nil {
		return "", time.Time{}, err
	}

	token, err := conf.TokenSource(oauth2.NoContext).Token()
	if err != nil {
		return "", time.Time{}, err
	}

	return token.AccessToken, token.Expiry, nil
}

//
type GCPTokenResponse struct {
	AccessToken string         `json:"access_token"`
	ExpiresIn   *time.Duration `json:"expires_in"`
	TokenType   string         `json:"token_type"`
}

//
func gcpTokenFromMetadata() (string, time.Time, error) {

	req, err := http.NewRequest("GET", gcp_metadata_url, nil)
	if err != nil {
		return "", time.Time{}, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Metadata-Flavor", "Google")

	client := &http.Client{}
	start := time.Now()

	resp, err := client.Do(req)
	if err != nil {
		return "", time.Time{}, err
	}

	defer resp.Body.Close()

	var respToken GCPTokenResponse

	err = json.NewDecoder(resp.Body).Decode(&respToken)
	if err != nil {
		return "", time.Time{}, err
	}

	expiry := start.Add(time.Second * *respToken.ExpiresIn)

	return respToken.AccessToken, expiry, nil
}
