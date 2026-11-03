import boto3
import sys

def test_localcloud():
    print("Testing Boto3 compatibility against LocalCloud (http://localhost:4566)...")
    session = boto3.Session(
        aws_access_key_id="test",
        aws_secret_access_key="test",
        region_name="us-east-1"
    )

    # 1. STS
    sts = session.client("sts", endpoint_url="http://localhost:4566")
    identity = sts.get_caller_identity()
    print("✓ STS GetCallerIdentity passed:", identity["Arn"], "Account:", identity["Account"])
    assert identity["Account"] == "000000000000"

    # 2. S3
    s3 = session.client("s3", endpoint_url="http://localhost:4566")
    buckets = s3.list_buckets()
    print("✓ S3 ListBuckets passed, found", len(buckets.get("Buckets", [])), "buckets.")

    # 3. SQS
    sqs = session.client("sqs", endpoint_url="http://localhost:4566")
    queues = sqs.list_queues()
    print("✓ SQS ListQueues passed, found", len(queues.get("QueueUrls", [])), "queues.")

    # 4. IAM
    iam = session.client("iam", endpoint_url="http://localhost:4566")
    users = iam.list_users()
    print("✓ IAM ListUsers passed, found", len(users.get("Users", [])), "users.")

    print("\nALL BOTO3 COMPATIBILITY TESTS PASSED!")

if __name__ == "__main__":
    test_localcloud()
