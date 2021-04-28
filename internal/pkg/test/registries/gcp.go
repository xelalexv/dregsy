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
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"testing"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	gcrauth "github.com/google/go-containerregistry/pkg/authn"
	gcrname "github.com/google/go-containerregistry/pkg/name"
	gcrgoogle "github.com/google/go-containerregistry/pkg/v1/google"
	gcrremote "github.com/google/go-containerregistry/pkg/v1/remote"

	"github.com/xelalexv/dregsy/internal/pkg/test"
)

//
type manifest struct {
	Digest string
	Info   gcrgoogle.ManifestInfo
}

//
func SkipIfGCPNotConfigured(t *testing.T, gcr, gar bool) {

	var missing []string

	if gcr && os.Getenv(test.EnvGCRProject) == "" {
		missing = append(missing, test.EnvGCRProject)
	}
	if gar && os.Getenv(test.EnvGARProject) == "" {
		missing = append(missing, test.EnvGARProject)
	}

	creds := os.Getenv(test.EnvGCPCreds)
	if creds == "" {
		missing = append(missing, test.EnvGCPCreds)
	}
	if len(missing) > 0 {
		t.Skipf(
			"skipping, GCR/GAR not configured, missing these environment variables: %v",
			missing)
	}
	if _, err := os.Stat(creds); err != nil {
		t.Skipf("skipping, GCP credentials file not accessible: %v", err)
	}
}

//
func EmptyGCRRepo(t *testing.T, p *test.Params) {
	repo, err := gcrname.NewRepository(
		fmt.Sprintf("%s/%s/%s", p.GCRHost, p.GCRProject, p.GCRImage))
	if err != nil {
		t.Fatal(err)
	}
	emptyGCPRepo(t, repo)
}

//
func EmptyGARRepo(t *testing.T, p *test.Params) {
	repo, err := gcrname.NewRepository(
		fmt.Sprintf("%s/%s/%s", p.GARHost, p.GARProject, p.GARImage))
	if err != nil {
		t.Fatal(err)
	}
	emptyGCPRepo(t, repo)
}

// this is very verbose, thought there'd be a better way, but...
// TODO: find out how to remove the repo
func emptyGCPRepo(t *testing.T, repo gcrname.Repository) {

	b, err := ioutil.ReadFile(os.Getenv(test.EnvGCPCreds))
	if err != nil {
		t.Fatal(err)
	}

	conf, err := google.JWTConfigFromJSON(
		b, "https://www.googleapis.com/auth/devstorage.full_control")
	if err != nil {
		t.Fatal(err)
	}

	token, err := conf.TokenSource(oauth2.NoContext).Token()
	if err != nil {
		t.Fatal(err)
	}
	auth := &gcrauth.Bearer{Token: token.AccessToken}

	tags, err := gcrgoogle.List(repo, gcrgoogle.WithAuth(auth))
	if err != nil {
		t.Fatal(err)
	}

	var manifests = make([]manifest, 0, len(tags.Manifests))
	for k, m := range tags.Manifests {
		manifests = append(manifests, manifest{k, m})
	}
	sort.Slice(manifests, func(i, j int) bool {
		return manifests[j].Info.Created.Before(manifests[i].Info.Created)
	})

	for _, man := range manifests {
		for _, tag := range man.Info.Tags { // first delete all tags
			if err := gcrremote.Delete(
				repo.Tag(tag), gcrremote.WithAuth(auth)); err != nil {
				t.Fatal(err)
			}
		}
		// now delete digest
		if err := gcrremote.Delete(
			repo.Digest(man.Digest), gcrremote.WithAuth(auth)); err != nil {
			t.Fatal(err)
		}
	}
}
