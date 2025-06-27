## Overview of vpc.go

The `vpc.go` file is part of a Platform infrastructure project that uses **Pulumi with Go** to provision AWS VPC networking infrastructure. This file creates a complete, production-ready VPC setup with multiple subnet types, NAT gateways, and security groups specifically designed for Kubernetes (EKS) workloads.

## Key Configuration Context

The VPC functions expect a configuration structure that includes:

```go
type Config struct {
	Environment   string   `yaml:"environment,omitempty"`
	Customer      string   `yaml:"customer,omitempty"`
	VpcCidr       string   `yaml:"vpc_cidr,omitempty"`
	Azs           []string `yaml:"availability_zones,omitempty"`
	eksVersion    string   `yaml:"eks_version,omitempty"`
	ProductDomain string   `yaml:"product_domain,omitempty"`
}
```

The actual configuration comes from YAML files like `us-east-1_dev.yaml`:
```yaml
environment: dev
customer: danieloa
vpc_cidr: "10.0.0.0/16"
availability_zones: [az1,az2,az3]
eks_version: "1.32"
product_domain: danieloa-vpc.example.com
```

## Function Breakdown

### 1. `CreateVPC()` - Main VPC Creation
- Creates the primary VPC with the specified CIDR block
- Enables DNS hostnames and DNS support for EKS compatibility
- Tags resources with customer, region, and product domain information

### 2. `CreateInternetGateway()` - Internet Connectivity
- Creates an Internet Gateway for public subnet internet access
- Attaches it to the VPC for outbound/inbound internet traffic

### 3. `CreateSubnets()` - Multi-Tier Subnet Architecture
Creates **four types of subnets**:
- **Public subnets**: For load balancers and NAT gateways
- **Platform subnets**: For the main EKS cluster workloads
- **Tekton subnets**: For CI/CD pipeline workloads (separate EKS cluster)
- **Database subnets**: For RDS instances

**Key Features**:
- Automatically adds Kubernetes-specific tags for EKS integration
- Tags subnets for AWS Load Balancer Controller discovery
- Adds Karpenter discovery tags for autoscaling
- Maps public IPs only for public subnets

### 4. `CreateNatGateway()` - Outbound Internet for Private Subnets
Creates **separate NAT Gateways** for isolation:
- **Platform NAT Gateway**: For platform subnet outbound traffic
- **Tekton NAT Gateway**: For tekton subnet outbound traffic  
- **Database subnets**: Share the platform NAT Gateway

This provides network isolation between different workload types.

### 5. `CreateRouteTables()` - Traffic Routing
Creates route tables and associations:
- **Public route table**: Routes traffic to Internet Gateway
- **Private route tables**: Route traffic to appropriate NAT Gateways
- Properly associates each subnet with its correct route table

### 6. `CreateSecurityGroups()` - Network Security
Creates security groups:
- **EKS Security Group**: Allows VPC traffic and HTTPS from anywhere
- **Database Security Group**: Allows PostgreSQL (5432) from VPC only

### 7. `SetupNetwork()` - Orchestration Function
This is the main entry point that:
1. Orchestrates all the above functions in the correct order
2. Handles dependencies between resources
3. Exports important resource IDs for use by other infrastructure stages
4. Returns all subnet IDs and security group IDs that EKS and RDS will need

## Architecture Pattern

This follows a **multi-stage infrastructure deployment** pattern:
- **Stage 1 (VPC)**: Networking foundation (this file)
- **Stage 2 (Core)**: EKS clusters, RDS databases, S3 buckets
- **Stage 3 (Apps)**: Applications and configurations

## Production-Ready Features

1. **Multi-AZ deployment** across availability zones
2. **Network isolation** with separate subnets for different workload types
3. **Kubernetes integration** with proper tagging for EKS and Karpenter
4. **Security groups** with least-privilege access
5. **Resource tagging** for cost tracking and management
6. **Error handling** with descriptive error messages

## Usage Context

This VPC infrastructure supports:
- **EKS clusters** for Platform and Tekton workloads
- **RDS databases** in private database subnets
- **AWS Load Balancer Controller** with proper subnet discovery
- **Karpenter autoscaling** with tagged subnets
- **Multi-region deployments** (region comes from AWS_REGION env var)

The exported values (VPC ID, subnet IDs, security group IDs) are used by subsequent infrastructure stages to deploy EKS clusters, RDS instances, and other AWS services within this VPC.I'll analyze the `vpc.go` file for you. Let me first read the complete file to understand its structure and functionality. ## VPC.go File Explanation

The vpc.go file is a comprehensive AWS VPC (Virtual Private Cloud) infrastructure module built with Pulumi in Go. This file creates a production-ready, multi-tier network architecture specifically designed for danieloa Platform deployments. Here's a detailed breakdown:

## 🏗️ **Architecture Overview**

This file implements a **sophisticated multi-subnet VPC architecture** with the following components:

### **1. Subnet Types & Purpose**
- **Public Subnets**: For load balancers and internet-facing resources
- **Platform Subnets**: For EKS clusters running platform services
- **Tekton Subnets**: For EKS clusters running CI/CD workloads
- **Database Subnets**: For RDS instances (isolated private subnets)

### **2. Core Functions Structure**

The file is organized into **modular functions** that create specific infrastructure components:

## 📋 **Function-by-Function Breakdown**

### **`CreateVPC()`** - Foundation
```go
func CreateVPC(ctx *pulumi.Context, config danieloaconfig.Config) (*ec2.Vpc, error)
```
- Creates the main VPC with customer-specific CIDR block
- Enables DNS hostnames and support for EKS requirements
- Applies consistent tagging for resource management

### **`CreateInternetGateway()`** - Internet Access
```go
func CreateInternetGateway(ctx *pulumi.Context, config danieloaconfig.Config, vpc *ec2.Vpc) (*ec2.InternetGateway, error)
```
- Creates Internet Gateway for public subnet internet access
- Attaches to the VPC for bidirectional internet connectivity

### **`CreateSubnets()`** - Network Segmentation
```go
func CreateSubnets(ctx *pulumi.Context, config danieloaconfig.Config, vpc *ec2.Vpc) (map[string][]*ec2.Subnet, error)
```
**This is the most sophisticated function** - it:
- Creates multiple subnet types based on configuration
- **Automatically applies EKS-specific tags**:
  - `kubernetes.io/role/elb` for public subnets
  - `kubernetes.io/role/internal-elb` for private subnets
  - `kubernetes.io/cluster/{cluster-name}` for cluster discovery
  - `karpenter.sh/discovery/{cluster-name}` for Karpenter node provisioning
- Maps public IP assignment for public subnets only

### **`CreateNatGateway()`** - Private Subnet Internet Access
```go
func CreateNatGateway(ctx *pulumi.Context, config danieloaconfig.Config, subnets map[string][]*ec2.Subnet, igw *ec2.InternetGateway) (map[string]*ec2.NatGateway, error)
```
**Advanced NAT Gateway strategy**:
- **Separate NAT Gateways** for Platform and Tekton subnets (network isolation)
- Database subnets share the Platform NAT Gateway (cost optimization)
- Creates Elastic IPs for each NAT Gateway
- Proper dependency management with Internet Gateway

### **`CreateRouteTables()`** - Traffic Routing
```go
func CreateRouteTables(ctx *pulumi.Context, config danieloaconfig.Config, vpc *ec2.Vpc, igw *ec2.InternetGateway, subnets map[string][]*ec2.Subnet, natGateways map[string]*ec2.NatGateway) (map[string]*ec2.RouteTable, error)
```
**Complex routing logic**:
- **Public Route Table**: Routes `0.0.0.0/0` to Internet Gateway
- **Private Route Tables**: Each subnet type gets its own route table
- **NAT Gateway Mapping**: Routes private subnet traffic to appropriate NAT Gateway
- **Subnet Association**: Automatically associates each subnet with correct route table

### **`CreateSecurityGroups()`** - Network Security
```go
func CreateSecurityGroups(ctx *pulumi.Context, config danieloaconfig.Config, vpc *ec2.Vpc) (map[string]*ec2.SecurityGroup, error)
```
Creates **production-ready security groups**:
- **EKS Security Group**: 
  - Allows all VPC traffic (required for EKS)
  - Allows HTTPS from internet
  - All outbound traffic allowed
- **Database Security Group**: 
  - PostgreSQL port (5432) from VPC only
  - Prevents direct internet access to databases

### **`SetupNetwork()`** - Orchestration Function
```go
func SetupNetwork(ctx *pulumi.Context, config danieloaconfig.Config) error
```
**The main orchestrator** that:
1. Calls all component functions in correct dependency order
2. Handles error propagation with detailed error messages
3. **Exports critical outputs** for other Pulumi stacks:
   - VPC ID
   - Subnet IDs by type (public, platform, tekton, database)
   - Security Group IDs (EKS, database)

## 🎯 **Key Design Principles**

### **1. EKS-Ready Architecture**
- All subnets have proper Kubernetes tags for EKS auto-discovery
- Supports Karpenter node provisioning
- Network load balancer support (internal and external)

### **2. Multi-Tenant Isolation**
- Separate NAT Gateways for different workload types
- Dedicated route tables per subnet type
- Security group isolation

### **3. Production Considerations**
- Proper dependency management between resources
- Comprehensive error handling with context
- Resource tagging for cost allocation and management
- Export values for cross-stack references

### **4. Cost Optimization**
- Database subnets share Platform NAT Gateway
- Efficient subnet distribution across availability zones

## 🔄 **Integration with Staged Deployment**

This VPC module is designed as **Stage 1** of the deployment pipeline:
- Creates foundational networking infrastructure
- Exports subnet IDs and security groups for **Stage 2** (EKS, RDS)
- Enables **Stage 3** applications to reference network resources

The modular design allows other stages to import these outputs using Pulumi's stack reference feature, maintaining clean separation of concerns while enabling complex multi-stage deployments.