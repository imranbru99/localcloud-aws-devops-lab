# LocalCloud — AWS-Compatible Local Cloud Learning Platform

> **A complete, local AWS-compatible cloud environment that runs on a single laptop or VPS. Learn real AWS hands-on with zero AWS account, zero credit card, and zero bill.**

Developed by **[Imran Ahmed](https://imrandev.bd)** (`me@imrandev.bd`).

---

## 🚀 Key Features

1. **100% AWS API Compatibility**:
   - Works directly with the official `aws` CLI, `boto3`, Terraform/OpenTofu, and AWS SDKs.
   - Simply point your endpoint to `http://localhost:4566` (or set `AWS_ENDPOINT_URL=http://localhost:4566`).
2. **Authentic AWS Management Console Web UI**:
   - Pixel-accurate AWS Console styling with Top Bar, Service Search (`Ctrl+K` / `Alt+S`), Region Selector, and Account Dropdown.
   - Interactive resource tables and create wizards for S3, IAM, EC2, SQS, SNS, DynamoDB, Lambda, KMS, and SSM.
   - **"Show equivalent CLI / SDK / Terraform"** code generator button on every action!
3. **Live API Inspector ("What just happened?")**:
   - Real-time Server-Sent Events (SSE) stream of every AWS API call hitting port 4566.
   - Displays caller principal, action, latency, payload JSON, IAM decision, and plain-English educational explanations.
4. **Guided Learning Hub & Automated Grading**:
   - 60+ guided hands-on lab scenarios mapped to AWS certifications (Cloud Practitioner, Solutions Architect, Developer, SysOps).
   - In-console **Auto-Checker** button that runs real API verification against LocalCloud to grade student progress live.
5. **IAM Policy Simulator & "Why Denied" Explainer**:
   - Step-by-step policy evaluation showing Deny > SCP > Boundary > Allow evaluation order.
6. **Billing & Cost Sandbox**:
   - Calculates simulated AWS monthly bills for created resources and explains real AWS pricing concepts.
7. **In-Browser CloudShell**:
   - Interactive terminal inside the console pre-configured to LocalCloud port 4566.
8. **One-Command Start & Doctor Diagnostics**:
   - `./localcloud doctor` checks CPU, RAM, and disk.
   - `./localcloud start` boots the entire local cloud in under 1 second.

---

## ⚡ Quickstart

### 1. Start LocalCloud
```bash
./localcloud start
```

Open the AWS Management Console in your browser:
👉 **[http://localhost:4566/console](http://localhost:4566/console)**

### 2. Configure Your Terminal (Dummy Credentials)
```bash
export AWS_ACCESS_KEY_ID="test"
export AWS_SECRET_ACCESS_KEY="test"
export AWS_DEFAULT_REGION="us-east-1"
```

### 3. Run Real AWS Commands
```bash
# Verify caller identity
aws --endpoint-url http://localhost:4566 sts get-caller-identity

# Create an S3 bucket and upload a file
aws --endpoint-url http://localhost:4566 s3 mb s3://my-learning-bucket
aws --endpoint-url http://localhost:4566 s3 cp myfile.txt s3://my-learning-bucket/
aws --endpoint-url http://localhost:4566 s3 ls s3://my-learning-bucket/

# Create an SQS queue and send a message
aws --endpoint-url http://localhost:4566 sqs create-queue --queue-name order-queue
aws --endpoint-url http://localhost:4566 sqs send-message \
    --queue-url http://localhost:4566/000000000000/order-queue \
    --message-body '{"orderId": "1001"}'

# Create an IAM user
aws --endpoint-url http://localhost:4566 iam create-user --user-name student-alice
```

---

## 🛠️ CLI Management Commands

```bash
./localcloud doctor     # Inspect system RAM, disk, and dependencies
./localcloud start      # Start gateway and web console
./localcloud status     # View running status and active services
./localcloud reset      # Cleanly reset all state to zero
./localcloud snapshot   # Export complete environment state archive
./localcloud stop       # Stop the local cloud
```

---

## 📚 Service Catalog

| Service | Implemented Operations | Wire Protocol |
|---|---|---|
| **STS** | `GetCallerIdentity`, `AssumeRole`, `GetSessionToken` | Query XML |
| **IAM** | `CreateUser`, `GetUser`, `ListUsers`, `CreateAccessKey`, `CreateRole`, `CreatePolicy`, `AttachUserPolicy`, `SimulateCustomPolicy` | Query XML |
| **S3** | `CreateBucket`, `ListBuckets`, `DeleteBucket`, `PutObject`, `GetObject`, `DeleteObject`, `ListObjectsV2`, `PutBucketVersioning` | REST-XML |
| **SQS** | `CreateQueue`, `GetQueueUrl`, `ListQueues`, `SendMessage`, `ReceiveMessage`, `DeleteMessage`, `GetQueueAttributes` | Dual: Query XML & JSON 1.0 |
| **SNS** | `CreateTopic`, `ListTopics`, `DeleteTopic`, `Subscribe`, `Unsubscribe`, `ListSubscriptions`, `Publish` | Query XML |
| **DynamoDB** | `CreateTable`, `DescribeTable`, `ListTables`, `DeleteTable`, `PutItem`, `GetItem`, `DeleteItem`, `Scan` | JSON 1.0 |
| **Lambda** | `CreateFunction`, `GetFunction`, `ListFunctions`, `DeleteFunction`, `Invoke` | REST-JSON |
| **KMS** | `CreateKey`, `DescribeKey`, `ListKeys`, `CreateAlias`, `Encrypt`, `Decrypt`, `GenerateDataKey` | JSON 1.1 (AES-256-GCM) |
| **SSM** | `PutParameter`, `GetParameter`, `GetParameters`, `GetParametersByPath`, `DeleteParameter` | JSON 1.1 |
| **Secrets Manager** | `CreateSecret`, `GetSecretValue`, `DescribeSecret`, `ListSecrets`, `DeleteSecret` | JSON 1.1 |
| **CloudWatch** | `PutMetricData`, `ListMetrics`, `PutMetricAlarm`, `DescribeAlarms` | Query XML |
| **EC2** | `RunInstances`, `DescribeInstances`, `TerminateInstances`, `CreateKeyPair`, `CreateSecurityGroup`, `CreateVpc` | EC2 Query |

---

## 📄 License & Attribution

Distributed under the **Apache-2.0 License**. See [LICENSE](file:///home/imran/project/aws/LICENSE) and [NOTICE](file:///home/imran/project/aws/NOTICE).

- **Author**: Imran Ahmed
- **Website**: [https://imrandev.bd](https://imrandev.bd)
- **Contact**: `me@imrandev.bd`
