package vpc

import (
	"fmt"
	"net"

	"github.com/apparentlymart/go-cidr/cidr"

	conf "github.com/danieloa/test-infra-go/config"

	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ec2"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func CreateVPC(ctx *pulumi.Context, config *conf.Config, tags pulumi.StringMap) (*ec2.Vpc, error) {
	// Create VPC with comprehensive configuration
	vpc, err := ec2.NewVpc(ctx, fmt.Sprintf("%s-vpc", config.Customer), &ec2.VpcArgs{
		CidrBlock:          pulumi.String(config.VpcCidr),
		EnableDnsHostnames: pulumi.Bool(true),
		EnableDnsSupport:   pulumi.Bool(true),
		Tags:               tags,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create VPC: %w", err)
	}
	return vpc, nil
}

func CreateInternetGateway(ctx *pulumi.Context, config *conf.Config, vpc *ec2.Vpc, tags pulumi.StringMap) (*ec2.InternetGateway, error) {
	// Create Internet Gateway
	igw, err := ec2.NewInternetGateway(ctx, fmt.Sprintf("%s-igw", config.Customer), &ec2.InternetGatewayArgs{
		VpcId: vpc.ID(),
		Tags:  tags,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Internet Gateway: %w", err)
	}
	return igw, nil
}

// GetNthSubnet splits a parentCIDR into n subnets and returns the y-th subnet as a string.
// y is 0-based. Returns error if n is not a power of 2 or y is out of range.
func GetNthSubnet(parentCIDR string, n, y int) (string, error) {
	_, ipnet, err := net.ParseCIDR(parentCIDR)
	if err != nil {
		return "", fmt.Errorf("invalid CIDR: %w", err)
	}
	if n <= 0 || y < 0 || y >= n {
		return "", fmt.Errorf("invalid n or y")
	}
	ones, bits := ipnet.Mask.Size()
	// n must be a power of 2, and the new prefix must not exceed bits
	newOnes := ones
	for pow := 1; pow < n; pow <<= 1 {
		newOnes++
	}
	if 1<<(newOnes-ones) != n || newOnes > bits {
		return "", fmt.Errorf("n must be a power of 2 and fit in the address space")
	}
	subnets, err := cidr.Subnet(ipnet, newOnes-ones, y)
	if err != nil {
		return "", err
	}
	return subnets.String(), nil
}

func CreateSubnets(ctx *pulumi.Context, config *conf.Config, vpc *ec2.Vpc, tags pulumi.StringMap) (map[string][]*ec2.Subnet, error) {
	// Create subnets for each type
	subnetList := []string{"public", "platform", "tekton", "database"}
	azList := []string{"use1-az1", "use1-az2", "use1-az3"} // Example AZs, replace with config.AZs if available
	subnets := make(map[string][]*ec2.Subnet)

	// Define subnet configurations
	for i, subnetType := range subnetList {
		subnet := make([]*ec2.Subnet, 0)

		for j, az := range azList {
			// Get the CIDR for the subnet
			cidr, err := GetNthSubnet(config.VpcCidr, len(subnetList), i*len(azList)+j)

			if err != nil {
				return nil, fmt.Errorf("failed to get CIDR for %s subnet in %s: %w", subnetType, az, err)
			}

			subnetSegment, err := ec2.NewSubnet(ctx, fmt.Sprintf("%s-%s-subnet-%d", config.Customer, subnetType, i), &ec2.SubnetArgs{
				VpcId:               vpc.ID(),
				CidrBlock:           pulumi.String(cidr),
				AvailabilityZone:    pulumi.String(az),
				MapPublicIpOnLaunch: pulumi.Bool(subnetType == "public"),
				Tags:                tags,
			})

			if err != nil {
				return nil, fmt.Errorf("failed to create %s subnet %d: %w", subnetType, i, err)
			}

			subnet = append(subnet, subnetSegment)
		}
		subnets[subnetType] = subnet
	}

	return subnets, nil
}

func CreateNatGateway(ctx *pulumi.Context, config *conf.Config, subnets map[string][]*ec2.Subnet, igw *ec2.InternetGateway, tags pulumi.StringMap) (map[string]*ec2.NatGateway, error) {
	// Create Elastic IPs and NAT Gateways with proper mapping
	natGateways := make(map[string]*ec2.NatGateway)
	if publicSubnets, exists := subnets["public"]; exists && len(publicSubnets) > 0 {
		// Create NAT Gateway for Platform subnets
		if _, exists := subnets["platform"]; exists {
			platformEip, err := ec2.NewEip(ctx, fmt.Sprintf("%s-platform-nat-eip", config.Customer), &ec2.EipArgs{
				Domain: pulumi.String("vpc"),
				Tags:   tags,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to create Platform NAT EIP: %w", err)
			}

			platformNatGw, err := ec2.NewNatGateway(ctx, fmt.Sprintf("%s-platform-nat-gateway", config.Customer), &ec2.NatGatewayArgs{
				AllocationId: platformEip.ID(),
				SubnetId:     publicSubnets[0].ID(),
				Tags:         tags,
			}, pulumi.DependsOn([]pulumi.Resource{igw}))
			if err != nil {
				return nil, fmt.Errorf("failed to create Platform NAT Gateway: %w", err)
			}
			natGateways["platform"] = platformNatGw
		}

		// Create NAT Gateway for Tekton subnets if they exist
		if _, exists := subnets["tekton"]; exists {
			tektonEip, err := ec2.NewEip(ctx, fmt.Sprintf("%s-tekton-nat-eip", config.Customer), &ec2.EipArgs{
				Domain: pulumi.String("vpc"),
				Tags:   tags,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to create Tekton NAT EIP: %w", err)
			}

			tektonNatGw, err := ec2.NewNatGateway(ctx, fmt.Sprintf("%s-tekton-nat-gateway", config.Customer), &ec2.NatGatewayArgs{
				AllocationId: tektonEip.ID(),
				SubnetId:     publicSubnets[1%len(publicSubnets)].ID(),
				Tags:         tags,
			}, pulumi.DependsOn([]pulumi.Resource{igw}))
			if err != nil {
				return nil, fmt.Errorf("failed to create Tekton NAT Gateway: %w", err)
			}
			natGateways["tekton"] = tektonNatGw
		}

		// Database subnets use the platform NAT Gateway
		if _, exists := subnets["database"]; exists {
			if platformNatGw, exists := natGateways["platform"]; exists {
				natGateways["database"] = platformNatGw
			}
		}
	}
	return natGateways, nil
}

func CreateRouteTables(ctx *pulumi.Context, config *conf.Config, vpc *ec2.Vpc, igw *ec2.InternetGateway, subnets map[string][]*ec2.Subnet, natGateways map[string]*ec2.NatGateway, tags pulumi.StringMap) (map[string]*ec2.RouteTable, error) {
	// Create Route Tables and routes
	routeTables := make(map[string]*ec2.RouteTable)

	// Public Route Table
	publicRT, err := ec2.NewRouteTable(ctx, fmt.Sprintf("%s-public-rt", config.Customer), &ec2.RouteTableArgs{
		VpcId: vpc.ID(),
		Tags:  tags,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create public route table: %w", err)
	}
	routeTables["public"] = publicRT

	// Route to Internet Gateway for public subnets
	_, err = ec2.NewRoute(ctx, fmt.Sprintf("%s-public-route", config.Customer), &ec2.RouteArgs{
		RouteTableId:         publicRT.ID(),
		DestinationCidrBlock: pulumi.String("0.0.0.0/0"),
		GatewayId:            igw.ID(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create public route: %w", err)
	}

	// Private Route Tables for each private subnet type with proper NAT Gateway mapping
	privateTypes := []string{"platform", "tekton", "database"}
	for _, subnetType := range privateTypes {
		if _, exists := subnets[subnetType]; exists {
			rt, err := ec2.NewRouteTable(ctx, fmt.Sprintf("%s-%s-rt", config.Customer, subnetType), &ec2.RouteTableArgs{
				VpcId: vpc.ID(),
				Tags:  tags,
			})
			if err != nil {
				return nil, fmt.Errorf("failed to create %s route table: %w", subnetType, err)
			}
			routeTables[subnetType] = rt

			// Route to the correct NAT Gateway for each private subnet type
			if natGw, exists := natGateways[subnetType]; exists {
				_, err = ec2.NewRoute(ctx, fmt.Sprintf("%s-%s-route", config.Customer, subnetType), &ec2.RouteArgs{
					RouteTableId:         rt.ID(),
					DestinationCidrBlock: pulumi.String("0.0.0.0/0"),
					NatGatewayId:         natGw.ID(),
				})
				if err != nil {
					return nil, fmt.Errorf("failed to create %s route: %w", subnetType, err)
				}
			}
		}
	}

	// Associate subnets with route tables
	for subnetType, typeSubnets := range subnets {
		rt := routeTables[subnetType]
		if rt == nil {
			continue
		}

		for i, subnet := range typeSubnets {
			_, err := ec2.NewRouteTableAssociation(ctx, fmt.Sprintf("%s-%s-rta-%d", config.Customer, subnetType, i), &ec2.RouteTableAssociationArgs{
				SubnetId:     subnet.ID(),
				RouteTableId: rt.ID(),
			})
			if err != nil {
				return nil, fmt.Errorf("failed to associate %s subnet %d with route table: %w", subnetType, i, err)
			}
		}
	}

	return routeTables, nil
}

func CreateSecurityGroups(ctx *pulumi.Context, config *conf.Config, vpc *ec2.Vpc, tags pulumi.StringMap) (map[string]*ec2.SecurityGroup, error) {
	// Create Security Groups
	securityGroups := make(map[string]*ec2.SecurityGroup)

	// Default security group for EKS
	eksSG, err := ec2.NewSecurityGroup(ctx, fmt.Sprintf("%s-eks-sg", config.Customer), &ec2.SecurityGroupArgs{
		Name:        pulumi.String(fmt.Sprintf("%s-eks-sg", config.Customer)),
		Description: pulumi.String("Security group for EKS cluster"),
		VpcId:       vpc.ID(),
		Ingress: ec2.SecurityGroupIngressArray{
			&ec2.SecurityGroupIngressArgs{
				Description: pulumi.String("All traffic from VPC"),
				FromPort:    pulumi.Int(0),
				ToPort:      pulumi.Int(65535),
				Protocol:    pulumi.String("tcp"),
				CidrBlocks:  pulumi.StringArray{pulumi.String(config.VpcCidr)},
			},
			&ec2.SecurityGroupIngressArgs{
				Description: pulumi.String("HTTPS"),
				FromPort:    pulumi.Int(443),
				ToPort:      pulumi.Int(443),
				Protocol:    pulumi.String("tcp"),
				CidrBlocks:  pulumi.StringArray{pulumi.String("0.0.0.0/0")},
			},
		},
		Egress: ec2.SecurityGroupEgressArray{
			&ec2.SecurityGroupEgressArgs{
				Description: pulumi.String("All outbound traffic"),
				FromPort:    pulumi.Int(0),
				ToPort:      pulumi.Int(0),
				Protocol:    pulumi.String("-1"),
				CidrBlocks:  pulumi.StringArray{pulumi.String("0.0.0.0/0")},
			},
		},
		Tags: tags,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create EKS security group: %w", err)
	}
	securityGroups["eks"] = eksSG

	// Database security group for RDS
	dbSG, err := ec2.NewSecurityGroup(ctx, fmt.Sprintf("%s-database-sg", config.Customer), &ec2.SecurityGroupArgs{
		Name:        pulumi.String(fmt.Sprintf("%s-database-sg", config.Customer)),
		Description: pulumi.String("Security group for RDS database"),
		VpcId:       vpc.ID(),
		Ingress: ec2.SecurityGroupIngressArray{
			&ec2.SecurityGroupIngressArgs{
				Description: pulumi.String("PostgreSQL from VPC"),
				FromPort:    pulumi.Int(5432),
				ToPort:      pulumi.Int(5432),
				Protocol:    pulumi.String("tcp"),
				CidrBlocks:  pulumi.StringArray{pulumi.String(config.VpcCidr)},
			},
		},
		Egress: ec2.SecurityGroupEgressArray{
			&ec2.SecurityGroupEgressArgs{
				Description: pulumi.String("All outbound traffic"),
				FromPort:    pulumi.Int(0),
				ToPort:      pulumi.Int(0),
				Protocol:    pulumi.String("-1"),
				CidrBlocks:  pulumi.StringArray{pulumi.String("0.0.0.0/0")},
			},
		},
		Tags: tags,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create database security group: %w", err)
	}
	securityGroups["database"] = dbSG

	return securityGroups, nil
}

func SetupNetwork(ctx *pulumi.Context, config *conf.Config, tags pulumi.StringMap) error {
	// Create VPC
	vpc, err := CreateVPC(ctx, config, tags)
	if err != nil {
		return fmt.Errorf("failed to create VPC: %w", err)
	}
	// Create Internet Gateway
	igw, err := CreateInternetGateway(ctx, config, vpc, tags)
	if err != nil {
		return fmt.Errorf("failed to create Internet Gateway: %w", err)
	}
	// Create subnets
	subnets, err := CreateSubnets(ctx, config, vpc, tags)
	if err != nil {
		return fmt.Errorf("failed to create subnets: %w", err)
	}
	// Create NAT Gateways
	natGateways, err := CreateNatGateway(ctx, config, subnets, igw, tags)
	if err != nil {
		return fmt.Errorf("failed to create NAT Gateways: %w", err)
	}
	// Create Route Tables
	_, err = CreateRouteTables(ctx, config, vpc, igw,
		subnets, natGateways, tags)
	if err != nil {
		return fmt.Errorf("failed to create route tables: %w", err)
	}
	// Create Security Groups
	securityGroups, err := CreateSecurityGroups(ctx, config, vpc, tags)
	if err != nil {
		return fmt.Errorf("failed to create security groups: %w", err)
	}

	// Export important values
	ctx.Export("vpcId", vpc.ID())
	if subnets["public"] != nil {
		publicSubnetIds := make(pulumi.StringArray, len(subnets["public"]))
		for i, subnet := range subnets["public"] {
			publicSubnetIds[i] = subnet.ID()
		}
		ctx.Export("publicSubnetIds", publicSubnetIds)
	}
	if subnets["platform"] != nil {
		platformSubnetIds := make(pulumi.StringArray, len(subnets["platform"]))
		for i, subnet := range subnets["platform"] {
			platformSubnetIds[i] = subnet.ID()
		}
		ctx.Export("platformSubnetIds", platformSubnetIds)
	}
	if subnets["tekton"] != nil {
		tektonSubnetIds := make(pulumi.StringArray, len(subnets["tekton"]))
		for i, subnet := range subnets["tekton"] {
			tektonSubnetIds[i] = subnet.ID()
		}
		ctx.Export("tektonSubnetIds", tektonSubnetIds)
	}
	if subnets["database"] != nil {
		databaseSubnetIds := make(pulumi.StringArray, len(subnets["database"]))
		for i, subnet := range subnets["database"] {
			databaseSubnetIds[i] = subnet.ID()
		}
		ctx.Export("databaseSubnetIds", databaseSubnetIds)
	}
	if securityGroups["eks"] != nil {
		ctx.Export("eksSecurityGroupId", securityGroups["eks"].ID())
	}
	if securityGroups["database"] != nil {
		ctx.Export("databaseSecurityGroupId", securityGroups["database"].ID())
	}
	ctx.Log.Info("VPC setup completed successfully", nil)
	// Return nil to indicate success

	return nil
}
