## IaC core feature requirements

Author: daniel ortega

### Multi-Stage Infrastructure Deployment

The deployment is organized into separated and clear stages (stage1, stage2, stage3), each responsible for a logical layer of infrastructure (e.g., VPC/networking, databases, EKS, S3, Opensearch, Helm charts, IAM, third-party integrations).

### Multi-Region Infra Deployment

This project should be region agnostic in the sense that we do not need to define the region anywhere but using whatever is set in AWS_REGION env var
Multi-Region: We leave the CICD process to reset AWS_REGION env var to process the multiple regions

### Avoid Resource Abstraction or Encapsulation, 

create functions that return Pulumi Objects (ie infrastructure resources)
it is ok to modularize components for clariry.

### Limited user configurable items

environment definition yaml file should be kept to a minimum. Ideally less than 30 lines.

### Automated Tagging

Implements auto-tagging for all resources to ensure consistent metadata and traceability across environments.

### Kubernetes and Helm Integration

Supports deploying and configuring Kubernetes clusters (EKS) and Helm charts for add-ons and controllers (e.g., Karpenter, AWS Load Balancer Controller, Datadog).

Consider EKS autopilot.

### Secrets and State Management

AWS secret manager as account's secret store
Automates creation of required secrets and Pulumi state buckets, supporting both initial setup and ongoing operations.

### Policy and IAM Management

Centralizes IAM and resource policies in dedicated modules, supporting secure and auditable access control.

### integration testing vs unit testing

test that resource are created and integrated ok within the env.
Rely on Pulumi SDK and AWS APIs for the validation of their parameters. if the CIDR is missing the Subnet will fail to create bc neither pulumi nor AWS accept empty cidrs for creating subnets. 

### CI/CD Integration

Includes scripts and helpers for automated testing and deployment in CI/CD pipelines.

### Extensible and Modular Design

New resources or stages can be added with minimal changes, thanks to modular structure and clear separation of concerns.

### Pulumi Native Features

Uses Pulumi’s stack outputs, configuration, and dependency management for robust, repeatable deployments.

