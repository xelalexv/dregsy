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

package skopeo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os/exec"
	"strings"

	log "github.com/sirupsen/logrus"
)

const defaultSkopeoBinary = "skopeo"
const defaultCertsBaseDir = "/etc/skopeo/certs.d"

var skopeoBinary string
var certsBaseDir string

//
func init() {
	skopeoBinary = defaultSkopeoBinary
	certsBaseDir = defaultCertsBaseDir
}

//
type tagList struct {
	Repository string   `json:"Repository"`
	Tags       []string `json:"Tags"`
}

//
func CertsDirForRepo(r string) string {
	return fmt.Sprintf("%s/%s", certsBaseDir, withoutPort(r))
}

//
func ListAllTags(ref, creds, certDir string, skipTLSVerify bool) (
	[]string, error) {

	cmd := []string{
		"list-tags",
	}

	if skipTLSVerify {
		cmd = append(cmd, "--tls-verify=false")
	}

	if creds != "" {
		cmd = append(cmd, fmt.Sprintf("--creds=%s", creds))
	}

	if certDir != "" {
		cmd = append(cmd, fmt.Sprintf("--cert-dir=%s", certDir))
	}

	cmd = append(cmd, "docker://"+ref)

	bufOut := new(bytes.Buffer)
	bufErr := new(bytes.Buffer)

	if err := runSkopeo(bufOut, bufErr, true, cmd...); err != nil {
		return nil,
			fmt.Errorf("error listing image tags for ref '%s': %s, %v",
				ref, bufErr.String(), err)
	}

	list, err := decodeTagList(bufOut.Bytes())
	if err != nil {
		return nil, err
	}
	return list.Tags, nil
}

//
func chooseOutStream(out io.Writer, verbose, isErrorStream bool) io.Writer {
	if verbose {
		if out != nil {
			return out
		}
		if isErrorStream {
			return log.StandardLogger().WriterLevel(log.ErrorLevel)
		}
		return log.StandardLogger().WriterLevel(log.InfoLevel)
	}
	return ioutil.Discard
}

//
func runSkopeo(outWr, errWr io.Writer, verbose bool, args ...string) error {

	cmd := exec.Command(skopeoBinary, args...)

	cmd.Stdout = chooseOutStream(outWr, verbose, false)
	cmd.Stderr = chooseOutStream(errWr, verbose, true)

	if err := cmd.Start(); err != nil {
		return err
	}

	if err := cmd.Wait(); err != nil {
		return err
	}

	return nil
}

//
func decodeTagList(tl []byte) (*tagList, error) {
	var ret tagList
	if err := json.Unmarshal(tl, &ret); err != nil {
		return nil, err
	}
	return &ret, nil
}

//
func withoutPort(repo string) string {
	ix := strings.Index(repo, ":")
	if ix == -1 {
		return repo
	}
	return repo[:ix]
}
