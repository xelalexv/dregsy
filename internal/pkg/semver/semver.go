package semver

import (
	"github.com/Masterminds/semver/v3"
	"github.com/xelalexv/dregsy/internal/pkg/log"
)

//
func MatchesSemverConstraint(constraint string, tag string) bool {
	if constraint == "" {
		return true
	}
	c, err := semver.NewConstraint(constraint)
	if err != nil {
		log.Info("Failed to parse semver constraint")
		log.Error(err)
		return false
	}
	v, err := semver.NewVersion(tag)
	if err != nil {
		log.Info("Tag is not parsable as semver")
		return false
	}

	return c.Check(v)
}
