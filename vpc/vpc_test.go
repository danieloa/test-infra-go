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
		Config: map[string]string{
			"stage":      "vpc",
			"stack":      "vpc",
			"aws:region": "us-east-1",
			"configFile": "deployment-configurations/us-east-1_dev.yaml",
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
		{"10.0.0.0/16", 3, 0, "10.0.0.0/18", false}, // n=3 rounds up to 4 (power of 2)
		{"10.0.0.0/16", 3, 1, "10.0.64.0/18", false},
		{"10.0.0.0/16", 12, 0, "10.0.0.0/20", false}, // n=12 rounds up to 16 (power of 2)
		{"10.0.0.0/16", 5, 5, "", true},              // y >= n should error
	}

	for _, tt := range tests {
		result, err := getNthSubnet(tt.parentCIDR, tt.n, tt.y)
		if tt.shouldErr && err == nil {
			t.Errorf("Expected error for input %v", tt)
		}
		if !tt.shouldErr && err != nil {
			t.Errorf("Unexpected error for input %v: %v", tt, err)
		}
		if !tt.shouldErr && result != tt.expected {
			t.Errorf("For input (CIDR=%s, n=%d, y=%d): Expected %s, got %s", tt.parentCIDR, tt.n, tt.y, tt.expected, result)
		}
	}
}
