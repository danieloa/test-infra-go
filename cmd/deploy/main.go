package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"

	danieloaConfig "github.com/danieloa/test-infra-go/config"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type CLIConfig struct {
	ConfigFile string
	Stage      string
	Action     string
}

func main() {
	var cfg CLIConfig

	var rootCmd = &cobra.Command{
		Use:   "deploy [stage]",
		Short: "danieloa Platform Infrastructure Deployment Tool",
		Long:  `A smart CLI tool for staged deployment of danieloa Platform infrastructure using Pulumi.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg.Stage = args[0]
			return deploy(cfg)
		},
	}

	var initCmd = &cobra.Command{
		Use:   "init",
		Short: "Initialize S3 backend",
		RunE: func(cmd *cobra.Command, args []string) error {
			return initBackend(cfg)
		},
	}

	// Add flags to root command
	rootCmd.Flags().StringVarP(&cfg.ConfigFile, "config", "c", "deployment-configurations/us-east-1_dev.yaml", "Config file path")
	rootCmd.Flags().StringVarP(&cfg.Action, "action", "a", "preview", "Pulumi action: up, down, preview, refresh")

	// Add flags to init command
	initCmd.Flags().StringVarP(&cfg.ConfigFile, "config", "c", "deployment-configurations/us-east-1_dev.yaml", "Config file path")

	rootCmd.AddCommand(initCmd)

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func loadConfig(filename string) (*danieloaConfig.Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config danieloaConfig.Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

func ensureAWSProfile() {
	os.Setenv("AWS_PROFILE", "danieloa-saas-vpc-install-1")
}

func checkAWSAuth() error {
	ensureAWSProfile()
	cmd := exec.Command("aws", "sts", "get-caller-identity")
	_, err := cmd.Output()
	if err != nil {
		fmt.Printf("❌ AWS authentication failed. Please run: aws sso login --profile danieloa-saas-vpc-install-1\n")
		return fmt.Errorf("AWS authentication failed: %w", err)
	}
	fmt.Printf("✅ AWS authentication successful\n")
	return nil
}

func initBackend(cfg CLIConfig) error {
	config, err := loadConfig(cfg.ConfigFile)
	if err != nil {
		return err
	}

	customerName := config.GetCustomerName()

	// Deploy to each region in config
	region := os.Getenv("AWS_REGION")

	fmt.Printf("🚀 Initializing S3 backend for customer: %s in primary region: %s\n", customerName, region)

	if err := checkAWSAuth(); err != nil {
		return err
	}

	// Create single S3 bucket in primary region for all stacks
	s3Bucket := fmt.Sprintf("danieloa-%s-%s-pulumi-state", config.Environment, config.Customer)

	fmt.Printf("📦 Creating S3 bucket: %s\n", s3Bucket)

	if err := createS3BucketIfNotExists(s3Bucket, region); err != nil {
		return err
	}

	// Configure Pulumi to use S3 backend
	s3BackendURL := fmt.Sprintf("s3://%s", s3Bucket)
	loginCmd := exec.Command("pulumi", "login", s3BackendURL)
	if err := loginCmd.Run(); err != nil {
		return fmt.Errorf("failed to configure Pulumi S3 backend: %w", err)
	}

	fmt.Printf("✅ S3 backend initialized: %s\n", s3BackendURL)
	fmt.Printf("   All region stacks will use this single state bucket\n")

	return nil
}

func createS3BucketIfNotExists(bucketName, region string) error {
	// Check if bucket exists
	checkCmd := exec.Command("aws", "s3api", "head-bucket", "--bucket", bucketName, "--region", region)
	if err := checkCmd.Run(); err == nil {
		fmt.Printf("✅ S3 bucket already exists: %s\n", bucketName)
		return nil
	}

	fmt.Printf("📦 Creating S3 bucket: %s\n", bucketName)

	var createCmd *exec.Cmd
	if region == "us-east-1" {
		createCmd = exec.Command("aws", "s3api", "create-bucket", "--bucket", bucketName, "--region", region)
	} else {
		createCmd = exec.Command("aws", "s3api", "create-bucket", "--bucket", bucketName, "--region", region,
			"--create-bucket-configuration", fmt.Sprintf("LocationConstraint=%s", region))
	}

	if err := createCmd.Run(); err != nil {
		return fmt.Errorf("failed to create S3 bucket: %w", err)
	}

	// Enable versioning
	versionCmd := exec.Command("aws", "s3api", "put-bucket-versioning", "--bucket", bucketName,
		"--versioning-configuration", "Status=Enabled", "--region", region)
	if err := versionCmd.Run(); err != nil {
		return fmt.Errorf("failed to enable versioning: %w", err)
	}

	return nil
}

func deploy(cfg CLIConfig) error {
	config, err := loadConfig(cfg.ConfigFile)
	if err != nil {
		return err
	}

	// Validate stage
	validStages := []string{"vpc", "core", "apps"}
	isValidStage := false
	for _, stage := range validStages {
		if cfg.Stage == stage {
			isValidStage = true
			break
		}
	}
	if !isValidStage {
		return fmt.Errorf("invalid stage '%s'. Valid stages: %v", cfg.Stage, validStages)
	}

	fmt.Printf("🚀 danieloa Platform Infrastructure Deployment\n")
	fmt.Printf("   Customer: %s\n", config.GetCustomerName())
	fmt.Printf("   Stage: %s\n", cfg.Stage)
	fmt.Printf("   Action: %s\n", cfg.Action)
	fmt.Printf("\n")

	if err := checkAWSAuth(); err != nil {
		return err
	}

	// Set environment variables
	os.Setenv("CONFIG_FILE", cfg.ConfigFile)
	fmt.Printf("Using config file: %s\n", cfg.ConfigFile)

	// Deploy to each region in config
	region := os.Getenv("AWS_REGION")

	fmt.Printf("🌎 Processing region: %s\n", region)

	stacksPerStage := map[string][]string{
		"vpc":  {"acm", "r53", "vpc"},
		"core": {"eks", "s3", "rds", "secrets", "kms"},
		"apps": {"app1", "app2", "app3"},
	}

	for _, stack := range stacksPerStage[cfg.Stage] {
		stackName := fmt.Sprintf("%s-%s-%s", config.GetCustomerName(), cfg.Stage+"-"+stack, region)
		fmt.Printf("🌎 stackName : %s\n", stackName)

		if err := deployToRegion(cfg, stackName, region); err != nil {
			return fmt.Errorf("deployment failed for region %s: %w", region, err)
		}

		fmt.Printf("✅ Region %s completed successfully!\n\n", region)

		// Show completion message
		switch cfg.Action {
		case "preview", "diff":
			fmt.Printf("✅ %s completed successfully for all regions!\n", cfg.Action)
		case "up":
			fmt.Printf("✅ %s stage deployed successfully to all regions!\n", cfg.Stage)
			// Show next steps
			switch cfg.Stage {
			case "vpc":
				fmt.Printf("\n🔗 Next: Deploy core stage with: deploy core\n")
			case "core":
				fmt.Printf("\n🔗 Next: Deploy apps stage with: deploy apps\n")
			case "apps":
				fmt.Printf("\n🎉 All stages deployed! Your infrastructure is ready.\n")
			}
		case "down":
			fmt.Printf("✅ %s stage destroyed successfully in all regions!\n", cfg.Stage)
		case "refresh":
			fmt.Printf("✅ Infrastructure state refreshed successfully in all regions!\n")
		}
	}

	return nil
}

func deployToRegion(cfg CLIConfig, stackName string, region string) error {
	// Select/create stack
	stackCmd := exec.Command("pulumi", "stack", "select", stackName, "--create")
	if err := stackCmd.Run(); err != nil {
		return fmt.Errorf("failed to select/create stack %s: %w", stackName, err)
	}

	// Set configuration values for this region
	configs := map[string]string{
		"stage":      cfg.Stage,
		"stackName":  stackName,
		"aws:region": region,
		"configFile": cfg.ConfigFile,
	}

	for key, value := range configs {
		configCmd := exec.Command("pulumi", "config", "set", key, value)
		if err := configCmd.Run(); err != nil {
			return fmt.Errorf("failed to set config %s: %w", key, err)
		}
	}

	// Deploy based on action
	var deployCmd *exec.Cmd
	switch cfg.Action {
	case "preview":
		deployCmd = exec.Command("pulumi", "'preview --diff'")
	case "up":
		deployCmd = exec.Command("pulumi", "up", "--no")
	case "down":
		deployCmd = exec.Command("pulumi", "destroy", "--yes")
	case "refresh":
		deployCmd = exec.Command("pulumi", "refresh", "--yes")
	default:
		return fmt.Errorf("unsupported action: %s", cfg.Action)
	}

	deployCmd.Stdout = os.Stdout
	deployCmd.Stderr = os.Stderr
	deployCmd.Env = os.Environ()

	return deployCmd.Run()
}
