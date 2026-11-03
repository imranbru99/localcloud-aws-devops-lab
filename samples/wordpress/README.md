# WordPress on LocalCloud (AWS Architecture Lab)

This directory contains a complete, working WordPress deployment running against **LocalCloud** using realistic AWS architecture principles:

1. **Amazon EC2 (Compute):** WordPress Apache + PHP runtime on port `8080`.
2. **Amazon RDS (Database):** MySQL 8.0 relational database on port `3306`.
3. **Amazon S3 (Object Storage):** `s3://wp-media-bucket` for offloading all media uploads.
4. **AWS IAM (Security):** IAM User with S3 access keys.

---

## One-Command Deployment

Run the deploy script:

```bash
./samples/wordpress/deploy.sh
```

This script will:
1. Provision the S3 bucket `s3://wp-media-bucket` in LocalCloud (`http://localhost:4566`).
2. Provision the RDS MySQL database metadata.
3. Launch the simulated EC2 web instance.
4. Start the WordPress container on `http://localhost:8080`.

---

## S3 Media Offload Configuration

To offload WordPress uploads directly to LocalCloud S3:

1. In WordPress (`http://localhost:8080`), install the **S3 Uploads** plugin or define in `wp-config.php`:

```php
define( 'S3_UPLOADS_BUCKET', 'wp-media-bucket' );
define( 'S3_UPLOADS_REGION', 'us-east-1' );
define( 'S3_UPLOADS_KEY', 'test' );
define( 'S3_UPLOADS_SECRET', 'test' );
define( 'S3_UPLOADS_ENDPOINT', 'http://localhost:4566' );
```

2. Every image uploaded to WordPress media library will be stored directly inside LocalCloud S3!

---

## Clean Up

To stop the containers:

```bash
docker compose -f samples/wordpress/docker-compose.yml down
```
