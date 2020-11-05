package semver

import (
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/xelalexv/dregsy/internal/pkg/log"
)

//
func MatchesSemverConstraint(constraint string, suffixes []string, tag string) bool {
	if constraint == "" {
		return true
	}
	c, err := semver.NewConstraint(constraint)
	if err != nil {
		log.Info("Failed to parse semver constraint")
		log.Error(err)
		return false
	}

	for _, suffix := range append(suffixes, "") {
		v, err := semver.NewVersion(strings.TrimSuffix(tag, suffix))
		if err != nil {
			continue
		}
		if c.Check(v) {
			return true
		}
	}

	return false
}
