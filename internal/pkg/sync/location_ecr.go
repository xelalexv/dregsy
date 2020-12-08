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
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"
	log "github.com/sirupsen/logrus"
)

//
func (l *Location) IsECR() bool {
	ecr, _, _ := l.GetECR()
	return ecr
}

//
func (l *Location) GetECR() (ecr bool, region, account string) {

	url := strings.Split(l.Registry, ".")

	ecr = (len(url) == 6 || len(url) == 7) && url[1] == "dkr" && url[2] == "ecr" &&
		url[4] == "amazonaws" && url[5] == "com" && (len(url) == 6 || url[6] == "cn")

	if ecr {
		region = url[3]
		account = url[0]
	} else {
		region = ""
		account = ""
	}

	return
}

//
func newECRAuthRefresher(l *Location, interval time.Duration) *ecrAuthRefresher {
	return &ecrAuthRefresher{loc: l, interval: interval}
}

//
type ecrAuthRefresher struct {
	loc      *Location
	interval time.Duration
	expiry   time.Time
}

//
func (rf *ecrAuthRefresher) refresh() error {

	if rf.loc == nil || rf.interval == 0 || time.Now().Before(rf.expiry) {
		return nil
	}

	_, region, account := rf.loc.GetECR()
	log.WithField("registry", rf.loc.Registry).Info("refreshing credentials")

	sess, err := session.NewSession()
	if err != nil {
		return err
	}

	svc := ecr.New(sess, &aws.Config{Region: aws.String(region)})
	input := &ecr.GetAuthorizationTokenInput{
		RegistryIds: []*string{aws.String(account)},
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

		user := strings.TrimSpace(split[0])
		pass := strings.TrimSpace(split[1])

		rf.loc.Auth = base64.StdEncoding.EncodeToString([]byte(
			fmt.Sprintf(`{"username": "%s", "password": "%s"}`, user, pass)))
		rf.expiry = time.Now().Add(rf.interval)

		return nil
	}

	return fmt.Errorf("no authorization data for '%s'", rf.loc.Registry)
}
