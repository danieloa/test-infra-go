package stages

import (
	"fmt"
	"strings"
)

// Stage represents deployment stages
// This is the ONLY pattern we're keeping from the original project
type Stage string

const (
	StageVPC  Stage = "vpc"
	StageCore Stage = "core"
	StageApps Stage = "apps"
)

// String returns the string representation of a stage
func (s Stage) String() string {
	return string(s)
}

// ParseStage parses a stage from string
func ParseStage(s string) (Stage, error) {
	stage := Stage(strings.ToLower(s))
	switch stage {
	case StageVPC, StageCore, StageApps:
		return stage, nil
	default:
		return "", fmt.Errorf("invalid stage: %s. Valid stages are: vpc, core, apps", s)
	}
}
