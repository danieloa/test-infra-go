# Core Infrastructure Module (`core/`)

This package implements the **Core** stage of the danieloa Platform infrastructure using [Pulumi](https://www.pulumi.com/) with Go. It provisions and exports core AWS resources such as EKS clusters and S3 buckets, following production-ready, modular, and extensible patterns.

---

## 📦 **What Does the Core Package Do?**

- **Provisions EKS Clusters:**  
  Deploys and exports IDs and endpoints for both platform and Tekton EKS clusters.
- **Creates S3 Buckets:**  
  Sets up general-purpose and FM (feature management) S3 buckets with appropriate ACLs, versioning, lifecycle rules, and (optionally) CORS.
- **Exports Outputs:**  
  All critical resource IDs and endpoints are exported for use by downstream stages (e.g., Apps).
- **Handles Tagging:**  
  All resources are tagged for traceability and cost allocation.

---

## 🗂️ **File Overview**

- [`core.go`](core.go): Main orchestration logic for EKS and S3 setup.
- [`eks.go`](eks.go): Private helpers for EKS cluster and IAM role creation.
- [`s3.go`](s3.go): S3 bucket creation logic, including lifecycle, versioning, and CORS.
- [`core_test.go`](core_test.go): Integration tests for EKS and S3 resource creation.

---

## ⚙️ **Key Functions**

### `SetupCoreInfrastructure(ctx, config, tags)`
- Orchestrates creation of EKS clusters and S3 buckets.
- Exports cluster IDs and endpoints.

### `setupEKSCluster(ctx, config, prefix, tags)`
- Creates an EKS cluster with all required IAM roles and policies.
- Used for both platform and Tekton clusters.

### `SetupS3Buckets(ctx, config, tags)`
- Creates general-purpose and FM S3 buckets.
- Handles CORS, ACL, versioning, and lifecycle rules.

### `createS3BucketUnified(ctx, bucketName, tags, withCors, acl, corsOrigins)`
- Internal helper for S3 bucket creation with flexible options.

---

## 🏷️ **Resource Tagging**

All resources are tagged with:
- `customer`
- `environment`
- `region`
- `product_domain`
- `stage`

---

## 🧪 **Testing**

- Integration tests are provided in [`core_test.go`](core_test.go).
- Tests validate that EKS clusters and S3 buckets are created and exported as expected.

---

## 🧩 **Extensibility**

- Add new core resources (e.g., RDS, Opensearch) by extending `SetupCoreInfrastructure`.
- Add new bucket types by updating the bucket lists in `core.go` or `s3.go`.
- All logic is modular and easy to extend.

---

## 🔗 **Integration with Other Stages**

- Exports outputs for use by the Apps stage and other stacks via Pulumi stack references.
- Designed to be called after the VPC stage, using exported subnet and security group IDs.

---

## 📖 **References**

- [Project Root README](../README.md) for overall architecture and usage.
- [Infra Core Feature Requirements](../docs/Infra_Core_Feature_Requirements.md) for design goals and requirements.

---

## 📝 **Example Usage**

```go
import (
    "github.com/danieloa/test-infra-go/core"
    // ...
)

err := core.SetupCoreInfrastructure(ctx, config, tags)
if err != nil {
    return err
}
```

---
🛠️ Contributing
Follow Go best practices for modularity and error handling.
Add integration tests for any new resources.
Keep resource tagging and output exports consistent.
