## 1. Project Purpose and Architecture

- The project is an **Infrastructure as Code (IaC)** solution using **Pulumi with Go** to provision AWS infrastructure for the "danieloa" platform.
- It is designed for **multi-stage** and **multi-region** deployments, with a focus on modularity, extensibility, and minimal configuration.

### Key Features (from `docs/Infra_Core_Feature_Requirements.md`)
- **Multi-Stage Deployment:** Stages are separated into VPC/networking, core services (EKS, RDS, S3), and applications.
- **Multi-Region Support:** Region is determined by the `AWS_REGION` environment variable; no hardcoding in config files.
- **Minimal User Configuration:** Environment YAML files are concise (see `deployment-configurations/`).
- **Automated Tagging:** All resources are tagged for traceability.
- **Kubernetes & Helm Integration:** Supports EKS clusters and Helm add-ons (Karpenter, AWS Load Balancer Controller, Datadog).
- **Secrets & State Management:** Uses AWS Secrets Manager and S3 for Pulumi state.
- **Centralized IAM:** IAM and policies are modularized.
- **CI/CD Ready:** Scripts and helpers for automated deployment.
- **Pulumi Native:** Uses stack outputs, config, and dependency management.

---

## 2. Project Structure

- **main.go:** Entry point for Pulumi, loads config, determines stage, and dispatches to the correct deployment logic.
- **cmd/deploy/main.go:** CLI tool for initializing backends, deploying stages, and managing stacks/regions.
- **config/config.go:** Loads and parses YAML configuration files.
- **vpc/vpc.go:** Implements all VPC, subnet, NAT gateway, route table, and security group logic.
- **stages/stages.go:** Defines and parses deployment stages.
- **deployment-configurations/**: Contains per-region, per-environment YAML config files.
- **Pulumi.yaml, Pulumi.env-us-east-1.yaml:** Pulumi project and stack configuration.

---

## 3. Configuration

### Example Config File (`deployment-configurations/us-east-1_dev.yaml`)
```yaml
environment: dev
customer: danieloa
vpc_cidr: "10.0.0.0/16"
availability_zones: [az1,az2,az3]
eks_version: "1.32"
product_domain: danieloa-vpc.example.com
```
- Minimal, region-agnostic, and focused on high-level parameters.

---

## 4. Deployment Stages

### Defined in `stages/stages.go` and orchestrated in `main.go`:
- **StageVPC:** Networking foundation (VPC, subnets, NAT, security groups).
- **StageCore:** Core services (EKS, RDS, S3) – placeholder for future logic.
- **StageApps:** Applications and configurations – placeholder for future logic.

---

## 5. VPC Module (`vpc/vpc.go` and `vpc/README.md`)

- **CreateVPC:** Main VPC with DNS, tagging.
- **CreateInternetGateway:** For public subnet internet access.
- **CreateSubnets:** Public, platform, tekton, and database subnets, with EKS/Karpenter tags.
- **CreateNatGateway:** Separate NATs for platform/tekton, shared for database.
- **CreateRouteTables:** Public/private route tables, correct associations.
- **CreateSecurityGroups:** EKS and database security groups.
- **SetupNetwork:** Orchestrates all above, exports resource IDs for later stages.

---

## 6. CLI Tool (`cmd/deploy/main.go`)

- **deploy [stage]:** Deploys a specific stage to the current region.
- **init:** Initializes S3 backend for Pulumi state.
- **Configurable via flags:** `--config`, `--action` (up, down, preview, refresh).
- **Handles AWS authentication, stack selection, and config propagation.**

---

## 7. Pulumi Integration

- **Pulumi.yaml:** Project metadata, config schema.
- **Pulumi.env-us-east-1.yaml:** Stack-specific config, including region and config file path.

---

## 8. Modularity and Extensibility

- Each stage is a function, easy to extend.
- New resources or stages can be added with minimal changes.
- Outputs from one stage are exported for use in subsequent stages.

---

## 9. Usage Flow

1. **Configure AWS credentials and set `AWS_REGION`.**
2. **Edit or create a config YAML in `deployment-configurations/`.**
3. **Initialize backend:** `deploy init`
4. **Deploy stages in order:** `deploy vpc`, then `deploy core`, then `deploy apps`
5. **Outputs (VPC ID, subnet IDs, etc.) are available for use in later stages or stacks.**

---

## 10. Example: VPC Stage

- Creates a production-ready, multi-AZ VPC with all required subnets, NATs, route tables, and security groups for EKS and RDS.
- All resources are tagged and exported for use by the core and apps stages.

---

## 11. References

- See `/docs/Infra_Core_Feature_Requirements.md` for a full list of requirements and design principles.
- See `/vpc/README.md` for a deep dive into the VPC module.

---

## 12. Makefile: Common Developer Tasks

A `Makefile` is provided to streamline common developer workflows:

### Build and Installation
| Target    | Description                                 |
|-----------|---------------------------------------------|
| `build`   | Build the deploy CLI tool to `bin/deploy`  |
| `install` | Install the deploy CLI tool to `$GOPATH/bin` |

### Testing and Quality
| Target    | Description                                 |
|-----------|---------------------------------------------|
| `test`    | Run Go and Pulumi integration tests        |
| `lint`    | Format and vet Go code                     |
| `all`     | Run `test`, `lint`, and `build`            |

### Infrastructure Management
| Target         | Description                                 |
|----------------|---------------------------------------------|
| `init`         | Initialize S3 backend for Pulumi state     |
| `deploy-vpc`   | Deploy VPC stage only                      |
| `deploy-core`  | Deploy Core stage only                     |
| `deploy-apps`  | Deploy Apps stage only                     |
| `deploy`       | Deploy all stages in sequence              |

### Preview Changes
| Target         | Description                                 |
|----------------|---------------------------------------------|
| `preview-vpc`  | Preview VPC stage changes (dry-run)        |
| `preview-core` | Preview Core stage changes (dry-run)       |
| `preview-apps` | Preview Apps stage changes (dry-run)       |

### Cleanup and Destruction
| Target         | Description                                 |
|----------------|---------------------------------------------|
| `destroy-vpc`  | Destroy VPC stage only                     |
| `destroy-core` | Destroy Core stage only                    |
| `destroy-apps` | Destroy Apps stage only                    |
| `destroy`      | Destroy all stages in reverse order        |
| `clean`        | Remove build/test artifacts                |

**Usage examples:**

```sh
# Development workflow
make test           # Run all tests
make lint           # Lint and format code
make build          # Build the CLI tool

# Infrastructure deployment
make init           # Initialize S3 backend (one-time setup)
make preview-vpc    # Preview VPC changes
make deploy-vpc     # Deploy VPC stage
make deploy-core    # Deploy Core stage
make deploy-apps    # Deploy Apps stage
make deploy         # Deploy all stages at once

# Cleanup
make destroy        # Destroy all infrastructure
make clean          # Remove build artifacts
```

This helps ensure code quality, repeatable deployments, and easy cleanup for all contributors.

---

This summary covers all the code and documentation relevant to explaining the project for README purposes. If you need more detail on any specific part (e.g., the VPC logic, CLI usage, or configuration), you can refer to the corresponding files listed above.Certainly! Here’s a clear explanation of your project, suitable for use in a README file:

---

## Project Overview

This project is an Infrastructure-as-Code (IaC) solution using Pulumi with Go to provision and manage AWS cloud infrastructure for the danieloa Platform. It is designed for production-ready, multi-stage, and multi-region deployments, with a focus on clarity, modularity, and minimal configuration.

### Key Features

- **Multi-Stage Deployment:** Infrastructure is organized into clear stages (vpc, `core`, `apps`), each responsible for a logical layer (networking, core services, applications).
- **Multi-Region Support:** The project is region-agnostic, using the `AWS_REGION` environment variable to determine the deployment region. CI/CD pipelines can deploy to multiple regions by updating this variable.
- **Minimal Configuration:** Environment YAML files are concise (less than 30 lines), containing only essential settings like environment, customer, VPC CIDR, and availability zones.
- **No Resource Abstraction:** Functions return Pulumi resource objects directly, avoiding unnecessary abstraction while allowing modularization for clarity.
- **Automated Tagging:** All resources are auto-tagged for traceability, including customer, region, product domain, and stage.
- **Kubernetes & Helm Integration:** Supports deploying EKS clusters and Helm charts for add-ons and controllers (e.g., Karpenter, AWS Load Balancer Controller, Datadog).
- **Secrets & State Management:** Uses AWS Secrets Manager for secrets and automates creation of Pulumi state buckets.
- **Centralized IAM & Policy Management:** IAM roles and policies are managed in dedicated modules for security and auditability.
- **Testing:** Emphasizes integration testing using Pulumi SDK and AWS APIs, relying on AWS validation for resource parameters.
- **CI/CD Ready:** Includes scripts and helpers for automated testing and deployment in CI/CD pipelines.
- **Extensible & Modular:** New resources or stages can be added with minimal changes due to the modular structure.
- **Pulumi Native Features:** Leverages Pulumi’s stack outputs, configuration, and dependency management for robust deployments.

### Architecture

- **Stage 1 (VPC):** Sets up networking (VPC, subnets, NAT gateways, route tables, security groups) with multi-AZ support and network isolation for different workloads.
- **Stage 2 (Core):** Deploys core services like EKS, RDS, and S3.
- **Stage 3 (Apps):** Deploys applications and configurations.

### Usage

- **Configuration:** Define your environment in a YAML file under deployment-configurations.
- **Deployment:** Use the CLI tool (main.go) to initialize backends and deploy specific stages.
- **Region Selection:** Set the `AWS_REGION` environment variable to control the deployment region.

---