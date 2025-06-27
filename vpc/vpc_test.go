package vpc
package vpc

import (
    "path/filepath"
    "testing"

    "github.com/pulumi/pulumi/pkg/v3/testing/integration"
)

func TestVpcInfra(t *testing.T) {
    cwd := filepath.Join("..", "..") // Adjust if needed to point to the project root

    integration.ProgramTest(t, &integration.ProgramTestOptions{
        Dir:                  cwd,
        Quick:                true,
        SkipRefresh:          true,
        ExpectRefreshChanges: true,
        Config: map[string]string{
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