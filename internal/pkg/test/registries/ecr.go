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

package registries

import (
	"os"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/aws/aws-sdk-go/service/ecrpublic"

	"github.com/xelalexv/dregsy/internal/pkg/sync"
	"github.com/xelalexv/dregsy/internal/pkg/test"
)

//
func SkipIfECRNotConfigured(t *testing.T, public bool) {
	var missing []string
	if os.Getenv(test.EnvAccessKeyID) == "" {
		missing = append(missing, test.EnvAccessKeyID)
	}
	if os.Getenv(test.EnvSecretAccessKey) == "" {
		missing = append(missing, test.EnvSecretAccessKey)
	}
	reg := test.EnvECRRegistry
	if public {
		reg = test.EnvECRPubRegistry
	}
	if os.Getenv(reg) == "" {
		missing = append(missing, reg)
	}
	if len(missing) > 0 {
		t.Skipf("skipping, ECR not configured, missing these environment variables: %v",
			missing)
	}
}

//
func RemoveECRRepo(t *testing.T, p *test.Params, public bool) {

	var loc *sync.Location
	if public {
		loc = &sync.Location{Registry: p.ECRPubRegistry}
	} else {
		loc = &sync.Location{Registry: p.ECRRegistry}
	}

	isEcr, pub, region, _ := loc.GetECR()

	if !isEcr {
		return
	}

	sess, err := session.NewSession()
	if err != nil {
		t.Fatal(err)
	}

	if pub {
		svc := ecrpublic.New(sess, &aws.Config{
			Region: aws.String(region),
		})

		inpDel := &ecrpublic.DeleteRepositoryInput{
			Force:          aws.Bool(true),
			RepositoryName: aws.String(p.ECRRepo),
		}

		if _, err := svc.DeleteRepository(inpDel); err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				if aerr.Code() == ecrpublic.ErrCodeRepositoryNotFoundException {
					return
				}
			}
			t.Fatal(err)
		}

	} else {
		svc := ecr.New(sess, &aws.Config{
			Region: aws.String(region),
		})

		inpDel := &ecr.DeleteRepositoryInput{
			Force:          aws.Bool(true),
			RepositoryName: aws.String(p.ECRRepo),
		}

		if _, err := svc.DeleteRepository(inpDel); err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				if aerr.Code() == ecr.ErrCodeRepositoryNotFoundException {
					return
				}
			}
			t.Fatal(err)
		}
	}
}
