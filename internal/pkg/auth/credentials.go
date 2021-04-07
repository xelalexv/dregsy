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

//
type Refresher interface {
	Refresh(creds *Credentials) error
}

//
func NewCredentialsFromBasic(username, password string) (*Credentials, error) {
	return &Credentials{username: username, password: password}, nil
}

//
func NewCredentialsFromToken(token string) (*Credentials, error) {
	return &Credentials{token: NewToken(token)}, nil
}

//
func NewCredentialsFromAuth(auth string) (*Credentials, error) {
	return decode(auth)
}

//
type Credentials struct {
	//
	username string
	password string
	//
	token     *Token
	refresher Refresher
	auther    Auther
}

//
func (c *Credentials) Username() string {
	return c.username
}

//
func (c *Credentials) Password() string {
	return c.password
}

//
func (c *Credentials) Auth() string {
	if c.auther == nil {
		return BasicAuth(c)
	}
	return c.auther(c)
}

//
func (c *Credentials) SetAuther(a Auther) {
	c.auther = a
}

//
func (c *Credentials) Token() *Token {
	return c.token
}

//
func (c *Credentials) SetToken(t *Token) {
	c.token = t
}

//
func (c *Credentials) SetRefresher(r Refresher) {
	c.refresher = r
}

//
func (c *Credentials) Refresh() error {
	if c.refresher == nil {
		return nil
	}
	return c.refresher.Refresh(c)
}
