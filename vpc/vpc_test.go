package vpc

import (
	"path/filepath"
	"testing"

	"github.com/pulumi/pulumi/pkg/v3/testing/integration"
)

func TestVpcInfra(t *testing.T) {
	cwd := filepath.Join("..") // vpc/ -> project root

	integration.ProgramTest(t, &integration.ProgramTestOptions{
		Dir:                  cwd,
		Quick:                true,
		SkipRefresh:          true,
		ExpectRefreshChanges: true,
		Config:               map[string]string{
			// Add minimal config values here if needed
		},
		ExtraRuntimeValidation: func(t *testing.T, stackInfo integration.RuntimeValidationStackInfo) {
			// Check that the expected outputs exist
			if _, ok := stackInfo.Outputs["vpcId"]; !ok {
				t.Errorf("Expected vpcId output")
			}
			if _, ok := stackInfo.Outputs["publicSubnetIds"]; !ok {
				t.Errorf("Expected publicSubnetIds output")
			}
			if _, ok := stackInfo.Outputs["platformSubnetIds"]; !ok {
				t.Errorf("Expected platformSubnetIds output")
			}
			if _, ok := stackInfo.Outputs["eksSecurityGroupId"]; !ok {
				t.Errorf("Expected eksSecurityGroupId output")
			}
		},
	})
}

func TestGetNthSubnet(t *testing.T) {
	tests := []struct {
		parentCIDR string
		n, y       int
		expected   string
		shouldErr  bool
	}{
		{"10.0.0.0/16", 4, 0, "10.0.0.0/18", false},
		{"10.0.0.0/16", 4, 1, "10.0.64.0/18", false},
		{"10.0.0.0/16", 3, 0, "", true}, // n not power of 2
	}

	for _, tt := range tests {
		result, err := getNthSubnet(tt.parentCIDR, tt.n, tt.y)
		if tt.shouldErr && err == nil {
			t.Errorf("Expected error for input %v", tt)
		}
		if !tt.shouldErr && result != tt.expected {
			t.Errorf("Expected %s, got %s", tt.expected, result)
		}
	}
}
