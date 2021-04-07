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
	"time"

	"golang.org/x/oauth2/jws"

	"github.com/dgrijalva/jwt-go"
	log "github.com/sirupsen/logrus"
)

//
func NewToken(raw string) *Token {
	ret := &Token{raw: raw}
	ret.decode()
	return ret
}

//
type Token struct {
	email    string
	scope    string
	audience string
	typ      string
	raw      string
	valid    bool
	//
	issue  time.Time
	expiry time.Time
}

//
func (t *Token) Email() string {
	return t.email
}

//
func (t *Token) Scope() string {
	return t.scope
}

//
func (t *Token) Audience() string {
	return t.audience
}

//
func (t *Token) Type() string {
	return t.typ
}

//
func (t *Token) Raw() string {
	return t.raw
}

//
func (t *Token) IsExpired() bool {
	return !t.IsValid() || time.Now().After(t.expiry)
}

//
func (t *Token) IsValid() bool {
	return t != nil && t.valid
}

//
func (t *Token) decode() {

	log.Debug("decoding token")

	// first try as JWS with Golang lib
	if claims, err := jws.Decode(t.raw); err == nil {
		t.email = claims.Iss
		t.scope = claims.Scope
		t.audience = claims.Aud
		t.typ = claims.Typ
		t.issue = time.Unix(claims.Iat, 0)
		t.expiry = time.Unix(claims.Exp, 0)
		t.valid = true
		log.Debugf("token decoded as JWS, valid until %v", t.expiry)
		return
	} else {
		log.Debugf("could not decode as JWS: %v", err)
	}

	// if that didn't work, try something else
	// FIXME: validate, not tried out
	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(t.raw, claims,
		func(token *jwt.Token) (interface{}, error) {
			return []byte("ok"), nil
		})
	if err == nil && token.Valid {
		t.email = getStringFromMapClaims(claims, "email")
		t.scope = getStringFromMapClaims(claims, "session_id")
		t.issue = getTimeFromMapClaims(claims, "iat")
		t.expiry = getTimeFromMapClaims(claims, "exp")
		t.valid = true
	}

	log.Debug("not a valid token")
	t.valid = false
}

//
func getTimeFromMapClaims(cm jwt.MapClaims, key string) time.Time {
	if val, ok := cm[key]; ok {
		if t, ok := val.(int64); ok {
			return time.Unix(t, 0)
		}
	}
	return time.Time{}
}

//
func getStringFromMapClaims(cm jwt.MapClaims, key string) string {
	if val, ok := cm[key]; ok {
		if ret, ok := val.(string); ok {
			return ret
		}
	}
	return ""
}
