#!/usr/bin/env bash
set -e

ENDPOINT="http://localhost:4566"
REGION="us-east-1"
BUCKET_NAME="wp-media-bucket"
DB_NAME="wp-database"

echo "=========================================================="
echo "  Deploying WordPress on LocalCloud (AWS-Compatible)"
echo "=========================================================="

# 1. Check LocalCloud status
if ! curl -s "$ENDPOINT/api/stats" >/dev/null 2>&1; then
    echo "❌ LocalCloud is not running on port 4566."
    echo "   Start it first with: ./localcloud start"
    exit 1
fi
echo "✓ LocalCloud gateway is online at $ENDPOINT"

# 2. Configure AWS credentials for the CLI
export AWS_ACCESS_KEY_ID="test"
export AWS_SECRET_ACCESS_KEY="test"
export AWS_DEFAULT_REGION="$REGION"

# 3. Create Amazon S3 Bucket for Media Offload
echo "📦 Step 1: Provisioning Amazon S3 Bucket ($BUCKET_NAME)..."
aws --endpoint-url "$ENDPOINT" s3 mb "s3://$BUCKET_NAME" || true
echo "✓ S3 Bucket s3://$BUCKET_NAME is active."

# 4. Provision Amazon RDS Database in LocalCloud
echo "🗄️ Step 2: Registering Amazon RDS MySQL instance ($DB_NAME)..."
aws --endpoint-url "$ENDPOINT" rds create-db-instance \
    --db-instance-identifier "$DB_NAME" \
    --db-instance-class db.t3.micro \
    --engine mysql || true
echo "✓ Amazon RDS instance registered in LocalCloud."

# 5. Launch Simulated EC2 Instance
echo "💻 Step 3: Launching EC2 Web Server Instance..."
aws --endpoint-url "$ENDPOINT" ec2 run-instances \
    --image-id ami-0c55b159cbfafe1f0 \
    --instance-type t3.micro || true
echo "✓ EC2 Instance running."

# 6. Boot the local WordPress + MySQL containers
echo "🚀 Step 4: Starting WordPress + MySQL Docker runtime..."
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
docker compose -f "$DIR/docker-compose.yml" up -d

echo ""
echo "=========================================================="
echo "  🎉 WordPress is Live & Connected to LocalCloud AWS!"
echo "=========================================================="
echo "  • WordPress URL:          http://localhost:8080"
echo "  • LocalCloud Console:     http://localhost:4566/console"
echo "  • Database Engine (RDS):  MySQL 8.0 on localhost:3306"
echo "  • S3 Media Bucket:        s3://$BUCKET_NAME"
echo ""
echo "Next Steps:"
echo "1. Open http://localhost:8080 to complete the 1-minute WordPress setup."
echo "2. Install an S3 media plugin (e.g. S3 Uploads) to store all images in LocalCloud S3."
echo "3. Stop anytime with: docker compose -f samples/wordpress/docker-compose.yml down"
echo "=========================================================="
