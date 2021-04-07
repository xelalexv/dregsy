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

package test

import (
	"os"

	"github.com/xelalexv/dregsy/internal/pkg/auth"
)

//
const (
	// Docker setup
	EnvDockerHost = "DREGSY_TEST_DOCKERHOST"

	// ECR
	EnvAccessKeyID     = "AWS_ACCESS_KEY_ID"
	EnvSecretAccessKey = "AWS_SECRET_ACCESS_KEY"
	EnvECRRegistry     = "DREGSY_TEST_ECR_REGISTRY"
	EnvECRRepo         = "DREGSY_TEST_ECR_REPO"

	// GCR
	EnvGCPCreds   = "GOOGLE_APPLICATION_CREDENTIALS"
	EnvGCRHost    = "DREGSY_TEST_GCR_HOST"
	EnvGCRProject = "DREGSY_TEST_GCR_PROJECT"
	EnvGCRImage   = "DREGSY_TEST_GCR_IMAGE"

	// Dockerhub
	EnvDockerhubUser = "DREGSY_TEST_DOCKERHUB_USER"
	EnvDockerhubPass = "DREGSY_TEST_DOCKERHUB_PASS"
)

//
type Params struct {
	DockerHost    string
	ECRRegistry   string
	ECRRepo       string
	GCRHost       string
	GCRProject    string
	GCRImage      string
	DockerhubAuth string
	LocalAuth     string
}

//
func GetParams() *Params {

	ret := &Params{
		DockerHost:  os.Getenv(EnvDockerHost),
		ECRRegistry: os.Getenv(EnvECRRegistry),
		ECRRepo:     os.Getenv(EnvECRRepo),
		GCRHost:     os.Getenv(EnvGCRHost),
		GCRProject:  os.Getenv(EnvGCRProject),
		GCRImage:    os.Getenv(EnvGCRImage),
	}

	if creds, err := auth.NewCredentialsFromBasic(
		"anonymous", "anonymous"); err == nil {
		ret.LocalAuth = auth.BasicAuthJSON(creds)
	}

	user := os.Getenv(EnvDockerhubUser)
	pass := os.Getenv(EnvDockerhubPass)

	if user != "" && pass != "" {
		if creds, err := auth.NewCredentialsFromBasic(user, pass); err == nil {
			ret.DockerhubAuth = auth.BasicAuthJSON(creds)
		}
	}

	if ret.DockerHost == "" {
		ret.DockerHost = "tcp://127.0.0.1:2375"
	}

	if ret.ECRRepo == "" {
		ret.ECRRepo = "dregsy/test"
	}

	if ret.GCRHost == "" {
		ret.GCRHost = "eu.gcr.io"
	}
	if ret.GCRImage == "" {
		ret.GCRImage = "dregsy/test"
	}

	return ret
}
