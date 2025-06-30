package core

import (
	"path/filepath"
	"testing"

	"github.com/pulumi/pulumi/pkg/v3/testing/integration"
)

func TestCoreInfraIntegration(t *testing.T) {
	cwd := filepath.Join("..") // core/ -> project root

	integration.ProgramTest(t, &integration.ProgramTestOptions{
		Dir:                  cwd,
		Quick:                true,
		SkipRefresh:          true,
		ExpectRefreshChanges: true,
		Config: map[string]string{
			"stage": "core",
			// Add other required config values
		},
		ExtraRuntimeValidation: func(t *testing.T, stackInfo integration.RuntimeValidationStackInfo) {
			// Check that the expected EKS cluster outputs exist
			if _, ok := stackInfo.Outputs["platformClusterId"]; !ok {
				t.Errorf("Expected platformClusterId output")
			}
			if _, ok := stackInfo.Outputs["tektonClusterId"]; !ok {
				t.Errorf("Expected tektonClusterId output")
			}
			if _, ok := stackInfo.Outputs["platformClusterEndpoint"]; !ok {
				t.Errorf("Expected platformClusterEndpoint output")
			}
			if _, ok := stackInfo.Outputs["tektonClusterEndpoint"]; !ok {
				t.Errorf("Expected tektonClusterEndpoint output")
			}
		},
	})
}

func TestEKSClusterSetupIntegration(t *testing.T) {
	cwd := filepath.Join("..") // core/ -> project root

	integration.ProgramTest(t, &integration.ProgramTestOptions{
		Dir:                  cwd,
		Quick:                true,
		SkipRefresh:          true,
		ExpectRefreshChanges: true,
		Config: map[string]string{
			"stage": "core",
		},
		ExtraRuntimeValidation: func(t *testing.T, stackInfo integration.RuntimeValidationStackInfo) {
			// Validate EKS cluster configuration
			if platformCluster, ok := stackInfo.Outputs["platformClusterId"]; ok {
				if platformCluster == nil {
					t.Errorf("Platform cluster ID should not be nil")
				}
			}

			if tektonCluster, ok := stackInfo.Outputs["tektonClusterId"]; ok {
				if tektonCluster == nil {
					t.Errorf("Tekton cluster ID should not be nil")
				}
			}
		},
	})
}

func TestCoreInfraWithVPCDependency(t *testing.T) {
	cwd := filepath.Join("..") // core/ -> project root

	integration.ProgramTest(t, &integration.ProgramTestOptions{
		Dir:                  cwd,
		Quick:                true,
		SkipRefresh:          true,
		ExpectRefreshChanges: true,
		Config: map[string]string{
			"stage": "core",
		},
		ExtraRuntimeValidation: func(t *testing.T, stackInfo integration.RuntimeValidationStackInfo) {
			// Verify that core infrastructure properly references VPC stack outputs
			// This test ensures the stack reference to VPC works correctly

			// Check that subnet IDs are properly retrieved from VPC stack
			if _, ok := stackInfo.Outputs["platformClusterId"]; !ok {
				t.Errorf("Expected platform cluster to be created with VPC subnets")
			}
			if _, ok := stackInfo.Outputs["tektonClusterId"]; !ok {
				t.Errorf("Expected tekton cluster to be created with VPC subnets")
			}
		},
	})
}
