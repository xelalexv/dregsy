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
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	awsecr "github.com/aws/aws-sdk-go/service/ecr"
	awsecrpub "github.com/aws/aws-sdk-go/service/ecrpublic"

	"github.com/xelalexv/dregsy/internal/pkg/util"
)

//
func IsECR(registry string) (ecr, public bool, region, account string) {

	if strings.HasSuffix(registry, "public.ecr.aws") {
		ecr = true
		public = true
		if ix := strings.Index(registry, "@"); ix > -1 {
			account = registry[:ix]
			if ix = strings.Index(account, ":"); ix > -1 {
				region = account[ix+1:]
				account = account[:ix]
			}
		}
		return
	}

	url := strings.Split(registry, ".")

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
func newECR(registry, region, account string) ListSource {
	return &ecr{
		registry: registry,
		region:   region,
		account:  account,
	}
}

//
type ecr struct {
	registry string
	region   string
	account  string
}

//
func (e *ecr) Retrieve(maxItems int) ([]string, error) {

	log.Debug("ECR retrieving image list")

	svc, err := e.getService()
	if err != nil {
		return nil, fmt.Errorf("error getting ECR service: %v", err)
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
		return nil, fmt.Errorf("error listing ECR repositories: %v", err)
	}

	return ret, nil
}

//
func (e *ecr) Ping() error {
	svc, err := e.getService()
	if err != nil {
		return err
	}
	_, err = svc.DescribeRegistry(&awsecr.DescribeRegistryInput{})
	return err
}

//
func (e *ecr) getService() (*awsecr.ECR, error) {
	sess, err := session.NewSession()
	if err != nil {
		return nil, err
	}
	return awsecr.New(sess, &aws.Config{Region: aws.String(e.region)}), nil
}

//
func CreateECRTarget(ref, region, account string, public bool) error {

	_, repo, _ := util.SplitRef(ref)
	if len(repo) == 0 {
		return nil
	}

	sess, err := session.NewSession()
	if err != nil {
		return err
	}

	if public {
		return createECRPubTarget(sess, ref, repo, region, account)
	} else {
		return createECRTarget(sess, ref, repo, region, account)
	}
}

//
func createECRPubTarget(sess *session.Session, ref, repo, region, account string) error {

	svc := awsecrpub.New(sess, &aws.Config{
		Region: aws.String(region),
	})

	rp := repo
	if p := strings.SplitN(repo, "/", 2); len(p) > 1 {
		rp = p[1]
	}

	log.WithFields(log.Fields{"long": repo, "short": rp}).Debug("repo")

	inpDescr := &awsecrpub.DescribeRepositoriesInput{
		RegistryId:      aws.String(account),
		RepositoryNames: []*string{aws.String(rp)},
	}

	out, err := svc.DescribeRepositories(inpDescr)
	if err == nil && len(out.Repositories) > 0 {
		log.WithField("ref", ref).Info("ECR public target already exists")
		return nil
	}

	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() != awsecrpub.ErrCodeRepositoryNotFoundException {
				return err
			}
		} else {
			return err
		}
	}

	log.WithField("ref", ref).Info("creating ECR public target")
	inpCrea := &awsecrpub.CreateRepositoryInput{
		RepositoryName: aws.String(rp),
	}

	_, err = svc.CreateRepository(inpCrea)
	return err
}

//
func createECRTarget(sess *session.Session, ref, repo, region, account string) error {

	svc := awsecr.New(sess, &aws.Config{
		Region: aws.String(region),
	})

	inpDescr := &awsecr.DescribeRepositoriesInput{
		RegistryId:      aws.String(account),
		RepositoryNames: []*string{aws.String(repo)},
	}

	out, err := svc.DescribeRepositories(inpDescr)
	if err == nil && len(out.Repositories) > 0 {
		log.WithField("ref", ref).Info("ECR target already exists")
		return nil
	}

	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() != awsecr.ErrCodeRepositoryNotFoundException {
				return err
			}
		} else {
			return err
		}
	}

	log.WithField("ref", ref).Info("creating ECR target")
	inpCrea := &awsecr.CreateRepositoryInput{
		RepositoryName: aws.String(repo),
	}

	_, err = svc.CreateRepository(inpCrea)
	return err
}
