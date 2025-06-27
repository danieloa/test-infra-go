package main

import (
	"fmt"
	"os"

	conf "github.com/danieloa/test-infra-go/config"
	stages "github.com/danieloa/test-infra-go/stages"
	vpc "github.com/danieloa/test-infra-go/vpc"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	pulumiconfig "github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// Get configuration from Pulumi config
		cfg := pulumiconfig.New(ctx, "")

		// Get stage from config (required)
		stageStr := cfg.Get("stage")
		stage, err := stages.ParseStage(stageStr)
		if err != nil {
			return fmt.Errorf("invalid stage '%s': %w", stageStr, err)
		}

		region := cfg.Get("region") // Get AWS region from Pulumi config
		if region == "" {
			region = os.Getenv("AWS_REGION")
			ctx.Log.Info("using AWS_REGION environment variable as default region, please set it in Pulumi config", nil)
		}

		// Get config file path
		configFile := cfg.Get("configFile")
		if configFile == "" {
			configFile = "deployment-configurations/" + region + ".yaml"
		}

		// Debug info
		ctx.Log.Info(fmt.Sprintf("Loading config from: %s", configFile), nil)

		// Load infrastructure configuration
		infraConfig, err := conf.LoadConfig(configFile)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		// Create common tags
		tags := pulumi.StringMap{
			"Customer":      pulumi.String(infraConfig.Customer),
			"Region":        pulumi.String(region),
			"ManagedBy":     pulumi.String("pulumi"),
			"Project":       pulumi.String("danieloa-infrastructure"),
			"ProductDomain": pulumi.String(infraConfig.ProductDomain),
			"Stage":         pulumi.String(stage.String()),
		}

		ctx.Log.Info(fmt.Sprintf("Deploying %s stage for customer: %s", stage.String(), infraConfig.CustomerName), nil)

		// Execute deployment based on stage
		switch stage {
		case stages.StageVPC:
			return deployVPCStage(ctx, infraConfig, tags)
		case stages.StageCore:
			return deployCoreStage(ctx, infraConfig, tags)
		case stages.StageApps:
			return deployAppsStage(ctx, infraConfig, tags)
		default:
			return fmt.Errorf("unknown stage: %s", stage.String())
		}
	})
}

// deployVPCStage deploys VPC infrastructure
func deployVPCStage(ctx *pulumi.Context, config *conf.Config, tags pulumi.StringMap) error {
	ctx.Log.Info("Deploying VPC stage", nil)

	err := vpc.SetupNetwork(ctx, config)
	if err != nil {
		return fmt.Errorf("failed to create VPC: %w", err)
	}

	return nil
}

// deployCoreStage deploys core services (EKS, RDS, S3)
func deployCoreStage(ctx *pulumi.Context, config *conf.Config, tags pulumi.StringMap) error {
	ctx.Log.Info("Deploying Core stage", nil)

	//TODO: Implement EKS, RDS, S3 deployment logic

	ctx.Log.Info("Core stage deployment completed successfully", nil)
	return nil
}

// deployAppsStage deploys applications and configurations
func deployAppsStage(ctx *pulumi.Context, config *conf.Config, tags pulumi.StringMap) error {
	ctx.Log.Info("Deploying Apps stage", nil)
	// TODO: Implement application deployment logic
	ctx.Log.Info("Apps stage deployment completed successfully", nil)
	return nil
}
