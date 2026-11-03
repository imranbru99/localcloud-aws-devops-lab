# LocalCloud Architecture & Technical Specification

LocalCloud is a high-fidelity, self-hosted AWS-compatible learning platform designed to run locally on a single machine. It allows students, engineers, and certification candidates to practice real AWS cloud operations using the official AWS CLI, Boto3, Terraform/OpenTofu, and SDKs without incurring AWS bills or needing cloud credentials.

---

## Architecture Diagram

```
┌───────────────────────────────── LocalCloud Management Console ─────────────────────────────────┐
│  • Authentic AWS Console Shell (Search [Ctrl+K], Region Switcher, Account Dropdown, CloudShell) │
│  • Resource Dashboards & Creation Wizards (S3, IAM, EC2, SQS, SNS, DynamoDB, Lambda, KMS, SSM)  │
│  • Live API Inspector ("What just happened?" with real-time SSE stream & plain-English explainer)│
│  • Guided Labs (60+ scenarios with live automated grading & verification against :4566)         │
│  • Infrastructure as Code Generator ("Show equivalent CLI / Terraform / CloudFormation")        │
└─────────────────────────────────▲───────────────────────────────────────▲───────────────────────┘
                                  │                                       │
                                  │ HTTP / SSE / REST                     │
┌─────────────────────────────────┴───────────────────────────────────────┴───────────────────────┐
│                                Edge Gateway (Port 4566, Single Endpoint)                        │
│  1. Wire Protocol Parsers:                                                                      │
│     - AWS Query (STS, IAM, SQS, SNS, CloudWatch)                                               │
│     - AWS JSON 1.0 & 1.1 (DynamoDB, KMS, SSM, Secrets Manager, Events)                          │
│     - REST-XML (Amazon S3 buckets & objects)                                                   │
│     - REST-JSON (AWS Lambda functions & invocations)                                            │
│     - EC2 Query (Amazon EC2 compute instances & VPCs)                                          │
│  2. SigV4 Signature Verification:                                                               │
│     - Canonical Request & StringToSign HMAC-SHA256 calculation                                  │
│     - Support for standard development keys & credentials                                       │
│  3. IAM Evaluation Engine:                                                                      │
│     - Explicit Deny > SCP > Permission Boundary > Identity/Resource Allow > Implicit Deny       │
│     - Action wildcard matching (`s3:*`, `iam:Create*`) & ARN resource resolution                │
│  4. CloudTrail Audit & Event Pipeline:                                                          │
│     - In-memory event ring buffer + append-only JSONL event stream                              │
│     - Real-time broadcast to connected Console Live API Inspector clients                       │
└─────────────────────────────────▲───────────────────────────────────────────────────────────────┘
                                  │
                         Service Adapters
                                  │
┌───────────────┬─────────────────┼────────────────┬────────────────┬─────────────────────────────┐
│ Storage (S3)  │ Identity (IAM)  │ Messaging(SQS) │ Compute (EC2)  │ Database & Keys (Dynamo/KMS)│
│ Filesystem    │ Policy State    │ In-memory FIFO │ Container /    │ NoSQL Key-Value Tables      │
│ Blob Engine   │ & Credentials   │ & DLQ engine   │ MicroVM State  │ & AES-256-GCM Cryptography  │
└───────────────┴─────────────────┴────────────────┴────────────────┴─────────────────────────────┘
```

---

## 1. Edge Gateway & Protocol Engine

The gateway listens on port `4566` (the canonical local cloud port). It inspects incoming HTTP requests to route them dynamically to the appropriate service adapter:
- **`X-Amz-Target` Header**: Identifies JSON services like DynamoDB (`DynamoDB_20120810.CreateTable`), KMS (`TrentService.CreateKey`), SSM (`AmazonSSM.PutParameter`).
- **Authorization Header (SigV4)**: Identifies service target from the credential scope `<AccessKeyId>/<Date>/<Region>/<Service>/aws4_request`.
- **Query / Form Parameters**: Identifies `Action=<Operation>` for Query-based services (STS, IAM, SQS, SNS).
- **HTTP Path & Method**: Identifies S3 bucket/key operations and Lambda invocations.

---

## 2. IAM Evaluation Engine

LocalCloud implements the exact AWS policy evaluation order:
1. **Explicit Deny**: Any statement evaluating to `Deny` halts evaluation and denies access.
2. **Service Control Policies (SCPs)**: For organizational account boundaries.
3. **Permissions Boundaries**: Restricts the maximum permissions an identity can have.
4. **Identity & Resource Policies**: Evaluates attached user/role policies and S3 bucket policies.
5. **Implicit Deny**: If no statement explicitly allows the operation, the request is denied with a canonical `403 AccessDenied` AWS error.

---

## 3. Real Behavior & Data Directory

All state in LocalCloud is isolated under the directory `${LOCALCLOUD_DATA_DIR:-./data}`:
- S3 objects are stored on disk in `./data/s3/<bucket>/<key>`.
- CloudTrail management events are stored in `./data/cloudtrail/events.jsonl`.
- Running `./localcloud reset` cleanly wipes and recreates the state without affecting the host machine.
