package stages

import (
	"fmt"
	"strings"
)

// Stage represents deployment stages
// This is the ONLY pattern we're keeping from the original project
type Stage string
type Stack string

const (
	StageVPC     Stage = "vpc"
	StageCore    Stage = "core"
	StageApps    Stage = "apps"
	StackACM     Stack = "acm"
	StackR53     Stack = "r53"
	StackVPC     Stack = "vpc"
	StackEKS     Stack = "eks"
	StackS3      Stack = "s3"
	StackRDS     Stack = "rds"
	StackSecrets Stack = "secrets"
	StackKMS     Stack = "kms"
	StackApp1    Stack = "app1"
	StackApp2    Stack = "app2"
	StackApp3    Stack = "app3"
)

var (
	StacksForStageVPCS  = []Stack{StackACM, StackR53, StackVPC}
	StacksForStageCoreS = []Stack{StackEKS, StackS3, StackRDS, StackSecrets, StackKMS}
	StacksForStageAppsS = []Stack{StackApp1, StackApp2, StackApp3}
)

// String returns the string representation of a stage
func (s Stage) String() string {
	return string(s)
}

// ParseStage parses a stage from string
func ParseStage(stageStr string, stackStr string) (Stage, Stack, error) {
	stage := Stage(strings.ToLower(stageStr))
	stack := Stack(strings.ToLower(stackStr))

	switch stage {
	case StageVPC:
		switch stack {
		case StackACM, StackR53, StackVPC:
			return stage, stack, nil
		default:
			return "", "", fmt.Errorf("invalid stack '%s' for stage '%s'. Valid stacks are: acm, r53, vpc", stack, stage)
		}
	case StageCore:
		switch stack {
		case StackEKS, StackS3, StackRDS, StackSecrets, StackKMS:
			return stage, stack, nil
		default:
			return "", "", fmt.Errorf("invalid stack '%s' for stage '%s'. Valid stacks are: eks, s3, rds, secrets, kms", stack, stage)
		}
	case StageApps:
		switch stack {
		case StackApp1, StackApp2, StackApp3:
			return stage, stack, nil
		default:
			return "", "", fmt.Errorf("invalid stack '%s' for stage '%s'. Valid stacks are: app1, app2, app3", stack, stage)
		}
	default:
		return "", "", fmt.Errorf("invalid stack '%s' for stage '%s'", stack, stage)
	}
}
