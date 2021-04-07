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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

//
type Auther func(*Credentials) string

//
func BasicAuth(c *Credentials) string {
	if isEmpty(c) {
		return ""
	}
	return base64Encode(fmt.Sprintf("%s:%s", c.username, c.password))
}

//
func BasicAuthJSON(c *Credentials) string {
	if isEmpty(c) {
		return ""
	}
	return base64Encode(fmt.Sprintf(
		`{"username": "%s", "password": "%s"}`, c.username, c.password))
}

//
func isEmpty(c *Credentials) bool {
	return c == nil || (c.username == "" && c.password == "")
}

//
func base64Encode(auth string) string {
	return base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf(auth)))
}

//
type jsonCreds struct {
	User string `json:"username"`
	Pass string `json:"password"`
}

//
func decode(auth string) (*Credentials, error) {

	data, err := base64.StdEncoding.DecodeString(auth)
	if err != nil {
		return nil, err
	}

	ret := &Credentials{}

	crd := &jsonCreds{}
	if err := json.Unmarshal(data, crd); err != nil {
		ret.auther = BasicAuth
		parts := strings.SplitN(string(data), ":", 2)
		ret.username = parts[0]
		if len(parts) > 1 {
			ret.password = parts[1]
		}
	} else {
		ret.auther = BasicAuthJSON
		ret.username = crd.User
		ret.password = crd.Pass
	}

	return ret, nil
}
