#!/bin/bash
# =============================================================================
# LocalStack Initialization Script
# Creates AWS resources for integration testing
# =============================================================================

set -e

echo "Initializing LocalStack AWS resources..."

# Wait for LocalStack to be fully ready
sleep 2

# =============================================================================
# S3 Buckets
# =============================================================================
echo "Creating S3 buckets..."

awslocal s3 mb s3://servicepro-test-documents || true
awslocal s3 mb s3://servicepro-test-invoices || true
awslocal s3 mb s3://servicepro-test-exports || true

# Set bucket policies for public read (test only)
awslocal s3api put-bucket-policy --bucket servicepro-test-documents --policy '{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "PublicReadGetObject",
      "Effect": "Allow",
      "Principal": "*",
      "Action": "s3:GetObject",
      "Resource": "arn:aws:s3:::servicepro-test-documents/*"
    }
  ]
}'

# =============================================================================
# SES (Email)
# =============================================================================
echo "Setting up SES..."

# Verify email identities
awslocal ses verify-email-identity --email-address test@servicepro.test || true
awslocal ses verify-email-identity --email-address noreply@servicepro.test || true

# =============================================================================
# SNS Topics
# =============================================================================
echo "Creating SNS topics..."

awslocal sns create-topic --name servicepro-test-notifications || true
awslocal sns create-topic --name servicepro-test-alerts || true
awslocal sns create-topic --name servicepro-test-sms || true

# =============================================================================
# SQS Queues
# =============================================================================
echo "Creating SQS queues..."

awslocal sqs create-queue --queue-name servicepro-test-jobs || true
awslocal sqs create-queue --queue-name servicepro-test-emails || true
awslocal sqs create-queue --queue-name servicepro-test-dlq || true

# Subscribe queues to topics
NOTIFICATION_TOPIC_ARN=$(awslocal sns list-topics --query "Topics[?contains(TopicArn, 'notifications')].TopicArn" --output text)
JOBS_QUEUE_URL=$(awslocal sqs get-queue-url --queue-name servicepro-test-jobs --query 'QueueUrl' --output text)
JOBS_QUEUE_ARN=$(awslocal sqs get-queue-attributes --queue-url $JOBS_QUEUE_URL --attribute-names QueueArn --query 'Attributes.QueueArn' --output text)

awslocal sns subscribe --topic-arn $NOTIFICATION_TOPIC_ARN --protocol sqs --notification-endpoint $JOBS_QUEUE_ARN || true

echo "LocalStack initialization complete!"
