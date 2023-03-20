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

	log "github.com/sirupsen/logrus"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/aws/aws-sdk-go/service/ecrpublic"
)

//
func NewECRAuthRefresher(public bool, account, region string,
	interval time.Duration) Refresher {

	return &ecrAuthRefresher{
		public:   public,
		account:  account,
		region:   region,
		interval: interval,
	}
}

//
type ecrAuthRefresher struct {
	public   bool
	account  string
	region   string
	interval time.Duration
	expiry   time.Time
}

//
func (rf *ecrAuthRefresher) Refresh(creds *Credentials) error {

	log.WithFields(log.Fields{
		"public":   rf.public,
		"region":   rf.region,
		"interval": rf.interval,
		"expiry":   rf.expiry}).Debug("ECR auth refresh")

	if rf.account == "" || rf.region == "" ||
		rf.interval == 0 || time.Now().Before(rf.expiry) {
		log.Debug("no auth refresh required")
		return nil
	}

	sess, err := session.NewSession()
	if err != nil {
		return err
	}

	data, err := rf.getAuthData(sess)
	if err != nil {
		return err
	}

	for _, d := range data {

		output, err := base64.StdEncoding.DecodeString(d)
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
		rf.expiry = time.Now().Add(rf.interval)

		return nil
	}

	return fmt.Errorf("no authorization data")
}

//
func (rf *ecrAuthRefresher) getAuthData(sess *session.Session) ([]string, error) {

	var ret []string

	if rf.public {
		svc := ecrpublic.New(sess, aws.NewConfig().WithRegion(rf.region))
		output, err := svc.GetAuthorizationToken(nil)
		if err != nil {
			return nil, err
		}
		ret = append(ret, *output.AuthorizationData.AuthorizationToken)

	} else {
		svc := ecr.New(sess, &aws.Config{Region: aws.String(rf.region)})
		input := &ecr.GetAuthorizationTokenInput{
			RegistryIds: []*string{aws.String(rf.account)},
		}
		output, err := svc.GetAuthorizationToken(input)
		if err != nil {
			return nil, err
		}
		for _, data := range output.AuthorizationData {
			ret = append(ret, *data.AuthorizationToken)
		}
	}

	return ret, nil
}
