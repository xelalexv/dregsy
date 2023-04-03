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

	"github.com/xelalexv/dregsy/internal/pkg/util"
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
func CertsDirForRegistry(r string) string {
	return fmt.Sprintf("%s/%s", certsBaseDir, withoutPort(r))
}

//
func ListAllTags(ref, creds, certDir string, skipTLSVerify bool) ([]string, error) {

	ret, err := info([]string{"list-tags"}, ref, creds, certDir, skipTLSVerify)
	if err != nil {
		return nil,
			fmt.Errorf("error listing image tags for ref '%s': %v", ref, err)
	}

	list, err := decodeTagList(ret)
	if err != nil {
		return nil, err
	}
	return list.Tags, nil
}

//
func Inspect(ref, platform, format, creds, certDir string, skipTLSVerify bool) (
	string, error) {

	cmd := addPlatformOverrides([]string{"inspect"}, platform)
	if format != "" {
		cmd = append(cmd, fmt.Sprintf("--format=%s", format))
	}

	if insp, err := info(cmd, ref, creds, certDir, skipTLSVerify); err != nil {
		return "", fmt.Errorf(
			"error inspecting image for ref '%s': %v", ref, err)
	} else {
		return strings.TrimSpace(string(insp)), nil
	}
}

//
func info(cmd []string, ref, creds, certDir string, skipTLSVerify bool) (
	[]byte, error) {

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
		return nil, fmt.Errorf("%s, %v", bufErr.String(), err)
	}

	return bufOut.Bytes(), nil
}

//
func addPlatformOverrides(cmd []string, platform string) []string {

	if platform != "" {
		os, arch, variant := util.SplitPlatform(platform)
		if os != "" {
			cmd = append(cmd, fmt.Sprintf("--override-os=%s", os))
		}
		if arch != "" {
			cmd = append(cmd, fmt.Sprintf("--override-arch=%s", arch))
		}
		if variant != "" {
			cmd = append(cmd, fmt.Sprintf("--override-variant=%s", variant))
		}
	}

	return cmd
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
func withoutPort(registry string) string {
	ix := strings.Index(registry, ":")
	if ix == -1 {
		return registry
	}
	return registry[:ix]
}
