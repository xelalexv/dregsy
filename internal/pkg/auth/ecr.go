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
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"
)

//
func NewECRAuthRefresher(account, region string, interval time.Duration) Refresher {
	return &ecrAuthRefresher{
		account:  account,
		region:   region,
		interval: interval,
	}
}

//
func NewECRSession(region string, config *map[string]string) (*session.Session, error) {
	awsConfig := aws.Config{Region: aws.String(region), CredentialsChainVerboseErrors: aws.Bool(true)}
	sessionOptions := session.Options{
		Config: awsConfig,
		SharedConfigState: session.SharedConfigEnable,
	}
	if profile, ok := (*config)["aws-profile"]; ok {
		sessionOptions.Profile = profile
	}

	sess, err := session.NewSessionWithOptions(sessionOptions)
	if err != nil {
		return nil, err
	}

	return sess, nil
}

//
type ecrAuthRefresher struct {
	account  string
	region   string
	interval time.Duration
	expiry   time.Time
}

//
func (rf *ecrAuthRefresher) Refresh(creds *Credentials) error {
	if rf.account == "" || rf.region == "" || time.Now().Before(rf.expiry) {
		return nil
	}

	config := creds.Config()
	sess, err := NewECRSession(rf.region, config)
	if err != nil {
		return err
	}

	svc := ecr.New(sess)
	input := &ecr.GetAuthorizationTokenInput{
		RegistryIds: []*string{aws.String(rf.account)},
	}
	authToken, err := svc.GetAuthorizationToken(input)
	if err != nil {
		return err
	}

	for _, data := range authToken.AuthorizationData {

		output, err := base64.StdEncoding.DecodeString(*data.AuthorizationToken)
		if err != nil {
			return err
		}

		split := strings.Split(string(output), ":")
		if len(split) != 2 {
			return fmt.Errorf("failed to parse credentials")
		}

		creds.username = strings.TrimSpace(split[0])
		creds.password = strings.TrimSpace(split[1])
		creds.auther = BasicAuthJSON

		if rf.interval == 0 {
			rf.expiry = *data.ExpiresAt
		} else {
			rf.expiry = time.Now().Add(rf.interval)
		}
		return nil
	}

	return fmt.Errorf("no authorization data")
}
