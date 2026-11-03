package console

import (
	"github.com/imrandev/localcloud/services/rds"
	"github.com/imrandev/localcloud/services/s3"
)

type Lab struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Track       string   `json:"track"` // CLF, SAA, DVA, SOA
	Level       string   `json:"level"` // Beginner, Intermediate, Advanced
	Duration    string   `json:"duration"`
	Objective   string   `json:"objective"`
	Diagram     string   `json:"diagram"`
	ConsolePath []string `json:"console_path"`
	CLIPath     []string `json:"cli_path"`
	Hints       []string `json:"hints"`
	Solution    string   `json:"solution"`
	RealAWSCost string   `json:"real_aws_cost"`
}

type LabCheckResult struct {
	LabID   string   `json:"lab_id"`
	Passed  bool     `json:"passed"`
	Score   int      `json:"score"`
	Details []string `json:"details"`
}

func GetAllLabs() []Lab {
	return []Lab{
		// 0. Full-Stack AWS Web App
		{
			ID:        "lab-wordpress-aws",
			Title:     "Deploy Scalable WordPress on AWS (EC2 + RDS MySQL + S3)",
			Track:     "Solutions Architect Associate",
			Level:     "Intermediate",
			Duration:  "25 mins",
			Objective: "Provision a multi-tier WordPress site: an EC2 web server, an RDS MySQL relational database, and an S3 bucket for media offload.",
			Diagram:   "User ──► EC2 (WordPress :8080) ──► RDS MySQL (:3306)\n                 └──► S3 Bucket (wp-media-bucket)",
			ConsolePath: []string{
				"1. Go to Amazon S3 Console and create bucket 'wp-media-bucket'.",
				"2. Go to Amazon RDS Console and create database 'wp-database' (Engine: MySQL).",
				"3. Go to Amazon EC2 Console and launch a t3.micro instance.",
				"4. Run ./samples/wordpress/deploy.sh to launch the WordPress stack.",
				"5. Visit http://localhost:8080 to complete the WordPress web wizard.",
			},
			CLIPath: []string{
				"aws --endpoint-url http://localhost:4566 s3 mb s3://wp-media-bucket",
				"aws --endpoint-url http://localhost:4566 rds create-db-instance --db-instance-identifier wp-database --db-instance-class db.t3.micro --engine mysql",
				"aws --endpoint-url http://localhost:4566 ec2 run-instances --image-id ami-0c55b159cbfafe1f0 --instance-type t3.micro",
				"./samples/wordpress/deploy.sh",
			},
			Hints: []string{
				"In real AWS, database passwords and credentials are stored in AWS Secrets Manager.",
				"Use S3 Uploads plugin with endpoint http://localhost:4566 to offload media.",
			},
			Solution:    "S3 bucket 'wp-media-bucket' and RDS instance 'wp-database' created, and WordPress running on port 8080.",
			RealAWSCost: "$38.50 / month (EC2 t3.micro + RDS db.t3.micro + S3).",
		},
		// 1. Storage & Hosting
		{
			ID:        "lab-s3-website",
			Title:     "Host a Static Website on Amazon S3",
			Track:     "Solutions Architect Associate",
			Level:     "Beginner",
			Duration:  "15 mins",
			Objective: "Create an S3 bucket, upload index.html, configure bucket policy for public read, and verify web access.",
			Diagram:   "User Browser ──► S3 Bucket (static hosting) ──► index.html",
			ConsolePath: []string{
				"1. Open Amazon S3 Console.",
				"2. Click 'Create bucket' and name it 'my-static-web-bucket'.",
				"3. Upload index.html.",
				"4. Attach public read bucket policy.",
			},
			CLIPath: []string{
				"aws --endpoint-url http://localhost:4566 s3 mb s3://my-static-web-bucket",
				"echo '<h1>Welcome to LocalCloud</h1>' > index.html",
				"aws --endpoint-url http://localhost:4566 s3 cp index.html s3://my-static-web-bucket/index.html",
			},
			Hints:       []string{"Bucket names must be globally unique.", "Remember to verify the object key name."},
			Solution:    "Ensure bucket 'my-static-web-bucket' exists with an index.html object.",
			RealAWSCost: "$0.02 / month in AWS Free Tier.",
		},
		{
			ID:        "lab-s3-versioning",
			Title:     "S3 Versioning and Accidental Deletion Protection",
			Track:     "Cloud Practitioner",
			Level:     "Beginner",
			Duration:  "10 mins",
			Objective: "Enable S3 bucket versioning to protect against unintended overwrites and deletions.",
			Diagram:   "Object v1 ──► Object v2 (Newer Version) ──► Delete Marker",
			ConsolePath: []string{
				"1. Navigate to S3 > Buckets.",
				"2. Select bucket > Properties > Bucket Versioning.",
				"3. Enable versioning.",
			},
			CLIPath: []string{
				"aws --endpoint-url http://localhost:4566 s3api put-bucket-versioning --bucket my-backup-bucket --versioning-configuration Status=Enabled",
			},
			Hints:       []string{"Once enabled, versioning cannot be deleted, only suspended."},
			Solution:    "Run put-bucket-versioning on your target bucket.",
			RealAWSCost: "$0.023 / GB for each stored version.",
		},
		// 2. Identity & Security
		{
			ID:        "lab-iam-least-privilege",
			Title:     "IAM Least Privilege & User Access Keys",
			Track:     "Security Specialty / SAA",
			Level:     "Intermediate",
			Duration:  "20 mins",
			Objective: "Create an IAM user 'developer-alice', create programmatic access keys, and assign a custom least-privilege policy.",
			Diagram:   "Developer ──► AccessKeyId/Secret ──► IAM Policy Evaluation ──► Allowed AWS Service",
			ConsolePath: []string{
				"1. Open IAM Console.",
				"2. Click Users > Add users > 'developer-alice'.",
				"3. Create access credentials.",
				"4. Attach policy.",
			},
			CLIPath: []string{
				"aws --endpoint-url http://localhost:4566 iam create-user --user-name developer-alice",
				"aws --endpoint-url http://localhost:4566 iam create-access-key --user-name developer-alice",
			},
			Hints:       []string{"Never share secret access keys.", "Use IAM roles when working with EC2 or Lambda."},
			Solution:    "Create the user 'developer-alice' and verify with `iam get-user`.",
			RealAWSCost: "$0.00 (IAM is always free).",
		},
		{
			ID:        "lab-iam-assume-role",
			Title:     "Cross-Service Delegation with IAM Roles & STS AssumeRole",
			Track:     "Solutions Architect Associate",
			Level:     "Advanced",
			Duration:  "25 mins",
			Objective: "Configure an IAM Role with trust policy and assume it using STS to acquire temporary security credentials.",
			Diagram:   "Client ──► sts:AssumeRole ──► Temporary Token (1 hour) ──► Access to Target Resource",
			ConsolePath: []string{
				"1. Open IAM Console > Roles.",
				"2. Create role 'DataProcessorRole'.",
				"3. Test role assumption in CloudShell.",
			},
			CLIPath: []string{
				"aws --endpoint-url http://localhost:4566 sts assume-role --role-arn arn:aws:iam::000000000000:role/DataProcessorRole --role-session-name test-session",
			},
			Hints:       []string{"Temporary credentials expire automatically."},
			Solution:    "Verify assumed role credentials using `sts get-caller-identity`.",
			RealAWSCost: "$0.00.",
		},
		// 3. Decoupling & Serverless
		{
			ID:        "lab-sqs-decoupling",
			Title:     "Decoupling Applications with Amazon SQS",
			Track:     "Developer Associate",
			Level:     "Beginner",
			Duration:  "15 mins",
			Objective: "Create an SQS standard queue 'order-processing', send a payload, and receive it.",
			Diagram:   "Order API ──► SQS Queue ('order-processing') ──► Background Worker",
			ConsolePath: []string{
				"1. Open SQS Console.",
				"2. Click Create Queue > 'order-processing'.",
				"3. Send and receive messages in the console tab.",
			},
			CLIPath: []string{
				"aws --endpoint-url http://localhost:4566 sqs create-queue --queue-name order-processing",
				"aws --endpoint-url http://localhost:4566 sqs send-message --queue-url http://localhost:4566/000000000000/order-processing --message-body '{\"orderId\":101}'",
			},
			Hints:       []string{"SQS queues offer at-least-once message delivery."},
			Solution:    "Verify queue 'order-processing' exists and has messages.",
			RealAWSCost: "$0.40 per 1M requests after free tier.",
		},
		{
			ID:        "lab-sns-sqs-fanout",
			Title:     "Pub/Sub Fanout Architecture with SNS and SQS",
			Track:     "Solutions Architect Associate",
			Level:     "Intermediate",
			Duration:  "25 mins",
			Objective: "Create an SNS topic 'order-events', subscribe multiple SQS queues, and publish a broadcast message.",
			Diagram:   "Publisher ──► SNS Topic ──┬──► SQS Queue 1 (Billing)\n                         └──► SQS Queue 2 (Analytics)",
			ConsolePath: []string{
				"1. Open SNS Console > Create topic 'order-events'.",
				"2. Subscribe SQS queues to the topic.",
				"3. Publish message to topic.",
			},
			CLIPath: []string{
				"aws --endpoint-url http://localhost:4566 sns create-topic --name order-events",
			},
			Hints:       []string{"SNS topics push messages immediately to all subscribers."},
			Solution:    "Check SNS topic subscriptions.",
			RealAWSCost: "$0.50 per 1M SNS publishes.",
		},
		// 4. Serverless & Databases
		{
			ID:        "lab-dynamodb-crud",
			Title:     "NoSQL High Performance with Amazon DynamoDB",
			Track:     "Developer Associate",
			Level:     "Intermediate",
			Duration:  "20 mins",
			Objective: "Create a DynamoDB table 'Products' with partition key 'ProductID' (String) and execute PutItem and GetItem.",
			Diagram:   "Client ──► DynamoDB Table ('Products') ──► Single-digit millisecond latency items",
			ConsolePath: []string{
				"1. Open DynamoDB Console.",
				"2. Click 'Create table', Table name: 'Products', Partition key: 'ProductID'.",
				"3. Explore items tab to create item.",
			},
			CLIPath: []string{
				"aws --endpoint-url http://localhost:4566 dynamodb create-table --table-name Products --attribute-definitions AttributeName=ProductID,AttributeType=S --key-schema AttributeName=ProductID,KeyType=HASH",
				"aws --endpoint-url http://localhost:4566 dynamodb put-item --table-name Products --item '{\"ProductID\":{\"S\":\"P100\"},\"Name\":{\"S\":\"Mechanical Keyboard\"}}'",
			},
			Hints:       []string{"Primary key is mandatory for every item."},
			Solution:    "Verify table 'Products' contains items using `dynamodb scan`.",
			RealAWSCost: "25 GB free storage per month in AWS.",
		},
		{
			ID:        "lab-lambda-serverless",
			Title:     "Serverless Compute with AWS Lambda",
			Track:     "Developer Associate",
			Level:     "Intermediate",
			Duration:  "20 mins",
			Objective: "Create a Python Lambda function and invoke it to process event data.",
			Diagram:   "Trigger ──► AWS Lambda (Serverless Container) ──► CloudWatch Logs",
			ConsolePath: []string{
				"1. Open Lambda Console.",
				"2. Click 'Create function', Name: 'hello-world', Runtime: Python 3.12.",
				"3. Test invoke in the console.",
			},
			CLIPath: []string{
				"aws --endpoint-url http://localhost:4566 lambda create-function --function-name hello-world --runtime python3.12 --role arn:aws:iam::000000000000:role/lambda-role --handler index.handler",
				"aws --endpoint-url http://localhost:4566 lambda invoke --function-name hello-world response.json",
			},
			Hints:       []string{"Lambda scales automatically with requests."},
			Solution:    "Function exists and successfully returns status 200.",
			RealAWSCost: "1,000,000 free requests per month forever.",
		},
		// 5. Compute & Networking
		{
			ID:        "lab-ec2-keypair",
			Title:     "Amazon EC2 Instance Launch & Key Pair Management",
			Track:     "Solutions Architect Associate",
			Level:     "Beginner",
			Duration:  "15 mins",
			Objective: "Create an RSA Key Pair, configure Security Group with port 22/80, and launch an EC2 instance.",
			Diagram:   "SSH Client (with .pem key) ──► Security Group (port 22) ──► EC2 Instance",
			ConsolePath: []string{
				"1. Open EC2 Console.",
				"2. Network & Security > Key Pairs > Create key pair 'my-key'.",
				"3. Launch instances using t3.micro.",
			},
			CLIPath: []string{
				"aws --endpoint-url http://localhost:4566 ec2 create-key-pair --key-name my-key",
				"aws --endpoint-url http://localhost:4566 ec2 run-instances --image-id ami-0c55b159cbfafe1f0 --instance-type t3.micro --key-name my-key",
			},
			Hints:       []string{"Save private key file with chmod 400 permissions."},
			Solution:    "Verify EC2 instance state is running using `ec2 describe-instances`.",
			RealAWSCost: "750 hours/month free on t2/t3.micro.",
		},
		{
			ID:        "lab-kms-encryption",
			Title:     "Data Protection with AWS KMS & Envelope Encryption",
			Track:     "Security Specialty",
			Level:     "Advanced",
			Duration:  "25 mins",
			Objective: "Create a Customer Master Key (CMK) in KMS, generate a 256-bit data key, and encrypt sensitive data.",
			Diagram:   "Plaintext + Data Key ──► Ciphertext | KMS Encrypted Data Key (Envelope)",
			ConsolePath: []string{
				"1. Open KMS Console.",
				"2. Click 'Create Key' > Symmetric AES-256.",
				"3. Test encrypt and decrypt.",
			},
			CLIPath: []string{
				"aws --endpoint-url http://localhost:4566 kms create-key --description 'Production Database Encryption Key'",
			},
			Hints:       []string{"Envelope encryption protects both data at rest and key exposure."},
			Solution:    "CMK created and validated with `kms list-keys`.",
			RealAWSCost: "$1.00 / key / month in AWS.",
		},
	}
}

func CheckLab(labID string) LabCheckResult {
	res := LabCheckResult{
		LabID:   labID,
		Passed:  false,
		Score:   0,
		Details: []string{},
	}

	switch labID {
	case "lab-wordpress-aws":
		buckets := s3.GetBuckets()
		hasBucket := false
		for _, b := range buckets {
			if b.Name == "wp-media-bucket" {
				hasBucket = true
				break
			}
		}
		if hasBucket {
			res.Details = append(res.Details, "[PASS] Amazon S3 media offload bucket (wp-media-bucket) exists")
			res.Score += 50
		} else {
			res.Details = append(res.Details, "[FAIL] Missing S3 bucket 'wp-media-bucket'. Create it using: aws s3 mb s3://wp-media-bucket")
		}

		dbs := rds.GetDBInstances()
		hasDB := false
		for _, d := range dbs {
			if d.DBInstanceIdentifier == "wp-database" {
				hasDB = true
				break
			}
		}
		if hasDB {
			res.Details = append(res.Details, "[PASS] Amazon RDS MySQL instance (wp-database) exists")
			res.Score += 50
		} else {
			res.Details = append(res.Details, "[FAIL] Missing RDS database 'wp-database'. Provision it using: aws rds create-db-instance")
		}

		// Resource presence does not prove that WordPress is running or configured.
		res.Passed = false
		res.Details = append(res.Details, "Resource checks only. Manually verify WordPress installation, database connectivity and media upload before considering this lab complete.")
		return res
	default:
		res.Details = append(res.Details, "Not automatically verified. Review the lab steps and record evidence manually; no automatic checker is implemented for this lab.")
	}

	return res
}
