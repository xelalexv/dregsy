/*
	Copyright 2023 Alexander Vollschwitz <xelalex@gmx.net>

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

	log "github.com/sirupsen/logrus"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	awsecr "github.com/aws/aws-sdk-go/service/ecrpublic"
)

//
func newECRPub(registry, account string) ListSource {
	return &ecrpub{registry: registry, account: account}
}

//
type ecrpub struct {
	registry string
	account  string
}

//
func (e *ecrpub) Retrieve(maxItems int) ([]string, error) {

	log.Debug("ECRpublic retrieving image list")

	svc, err := e.getService()
	if err != nil {
		return nil, fmt.Errorf("error getting ECRpublic service: %v", err)
	}

	input := &awsecr.DescribeRepositoriesInput{
		RegistryId: aws.String(e.account),
		MaxResults: aws.Int64(100), // this is max page size
	}

	var ret []string

	if err := svc.DescribeRepositoriesPages(input,
		func(page *awsecr.DescribeRepositoriesOutput, lastPage bool) bool {
			for _, r := range page.Repositories {
				ret = append(ret, aws.StringValue(r.RepositoryName))
			}
			return maxItems <= 0 || len(ret) < maxItems
		}); err != nil {
		return nil, fmt.Errorf("error listing ECRpublic repositories: %v", err)
	}

	return ret, nil
}

//
func (e *ecrpub) Ping() error {
	svc, err := e.getService()
	if err != nil {
		return err
	}
	_, err = svc.DescribeRegistries(&awsecr.DescribeRegistriesInput{
		MaxResults: aws.Int64(1),
	})
	return err
}

//
func (e *ecrpub) getService() (*awsecr.ECRPublic, error) {
	sess, err := session.NewSession()
	if err != nil {
		return nil, err
	}
	return awsecr.New(sess), nil
}
