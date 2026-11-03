# LocalCloud Service API Coverage Matrix

This matrix tracks implemented operations, compatibility status, and implementation approach across LocalCloud services.

| Service | Protocol | Status | Implemented Operations | Notes |
|---|---|---|---|---|
| **STS** | Query | Full emulation | `GetCallerIdentity`, `AssumeRole`, `GetSessionToken` | Returns canonical XML with temporary session tokens and ARNs. |
| **IAM** | Query | Full emulation | `CreateUser`, `GetUser`, `ListUsers`, `DeleteUser`, `CreateAccessKey`, `ListAccessKeys`, `DeleteAccessKey`, `CreateRole`, `GetRole`, `ListRoles`, `DeleteRole`, `CreatePolicy`, `GetPolicy`, `ListPolicies`, `AttachUserPolicy`, `DetachUserPolicy`, `AttachRolePolicy`, `DetachRolePolicy`, `SimulateCustomPolicy`, `SimulatePrincipalPolicy` | Full policy evaluation engine supporting wildcards, boundaries, and statements. |
| **S3** | REST-XML | Real engine | `CreateBucket`, `ListBuckets`, `DeleteBucket`, `HeadBucket`, `PutObject`, `GetObject`, `DeleteObject`, `ListObjects`, `ListObjectsV2`, `HeadObject`, `PutBucketPolicy`, `GetBucketPolicy`, `DeleteBucketPolicy`, `PutBucketVersioning`, `GetBucketVersioning` | Direct local storage engine under `./data/s3/`. Computes MD5 checksums and ETags. |
| **SQS** | Query & JSON 1.0 | Full emulation | `CreateQueue`, `GetQueueUrl`, `ListQueues`, `SendMessage`, `ReceiveMessage`, `DeleteMessage`, `DeleteQueue`, `PurgeQueue`, `GetQueueAttributes` | Dual-mode parser supporting legacy XML Query and modern JSON 1.0 protocols. |
| **SNS** | Query | Full emulation | `CreateTopic`, `ListTopics`, `DeleteTopic`, `Subscribe`, `Unsubscribe`, `ListSubscriptions`, `Publish` | In-memory pub/sub engine with topic and subscription management. |
| **DynamoDB** | JSON 1.0 | Full emulation | `CreateTable`, `DescribeTable`, `ListTables`, `DeleteTable`, `PutItem`, `GetItem`, `DeleteItem`, `Scan` | Full primary key HASH/RANGE index tracking, attribute storage, and table lifecycle. |
| **Lambda** | REST-JSON | Simplified | `CreateFunction`, `GetFunction`, `ListFunctions`, `DeleteFunction`, `Invoke` | Stores function metadata and runs serverless invocation emulation. |
| **KMS** | JSON 1.1 | Real engine | `CreateKey`, `DescribeKey`, `ListKeys`, `CreateAlias`, `ListAliases`, `Encrypt`, `Decrypt`, `GenerateDataKey` | Real AES-256-GCM authenticated encryption and cryptographic key derivation. |
| **Secrets Manager** | JSON 1.1 | Full emulation | `CreateSecret`, `GetSecretValue`, `DescribeSecret`, `ListSecrets`, `PutSecretValue`, `DeleteSecret` | Encrypted secret storage with versioning and ARN tracking. |
| **SSM Parameter Store** | JSON 1.1 | Full emulation | `PutParameter`, `GetParameter`, `GetParameters`, `GetParametersByPath`, `DeleteParameter` | Standard & SecureString parameter hierarchies with automatic version incrementing. |
| **CloudWatch** | Query | Full emulation | `PutMetricData`, `ListMetrics`, `PutMetricAlarm`, `DescribeAlarms` | Custom metric ingestion, alarm definitions, and threshold monitoring. |
| **EC2** | EC2 Query | Simplified | `DescribeInstances`, `RunInstances`, `TerminateInstances`, `StopInstances`, `StartInstances`, `DescribeKeyPairs`, `CreateKeyPair`, `DescribeSecurityGroups`, `CreateSecurityGroup`, `AuthorizeSecurityGroupIngress`, `DescribeVpcs`, `CreateVpc`, `DescribeSubnets`, `CreateSubnet` | Generates RSA key pairs, security groups, VPCs, and containerized instance representations. |

---

## Simplification Disclosures (Honest Differences from Real AWS)
- **Multi-AZ**: In real AWS, Availability Zones reside in isolated data centers. In LocalCloud, AZs are logical namespaces on a single machine for architectural learning.
- **Billing**: LocalCloud is 100% free and offline. The Billing Sandbox provides educational price calculations based on real AWS public rate cards.
- **Data Durability**: Data is stored locally on the host disk under `./data`. Learners should use `./localcloud snapshot` to backup lab environments.
