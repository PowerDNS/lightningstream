#!/bin/bash

# PowerDNS Documentation Publishing Script
#
# This script uploads documentation to an S3 bucket and invalidates the CloudFront cache. This uses the AWS CLI.
#
# Environment Variables Required:
# - AWS_ACCESS_KEY_ID: The AWS access key ID
# - AWS_SECRET_ACCESS_KEY: The AWS secret access key
# - AWS_REGION: The AWS region where resources are located
# - AWS_S3_BUCKET_DOCS: The name of the S3 bucket for documentation
# - AWS_CLOUDFRONT_DISTRIBUTION_ID_DOCS: The CloudFront distribution ID
#
# Usage:
# ./publish.sh <SOURCE_PATH> [TARGET_DIR]

set -e  # Exit immediately if a command exits with a non-zero status

# Check if AWS CLI is installed
if ! command -v aws &> /dev/null; then
    echo "AWS CLI is not installed. Please install it and try again."
    exit 1
fi

# Function to get content type based on file extension
get_content_type() {
    case "${1##*.}" in
        html) echo "text/html" ;;
        css)  echo "text/css" ;;
        js)   echo "application/javascript" ;;
        json) echo "application/json" ;;
        png)  echo "image/png" ;;
        jpg|jpeg) echo "image/jpeg" ;;
        *)    echo "application/octet-stream" ;;
    esac
}

# Function to upload file or directory to S3
upload_to_s3() {
    local source_path="$1"
    local dest_dir="$2"

    if [ -d "$source_path" ]; then
        for file in "$source_path"/*; do
            if [ -d "$file" ]; then
                upload_to_s3 "$file" "${dest_dir}/$(basename "$file")"
            else
                upload_file_to_s3 "$file" "${dest_dir}"
            fi
        done
    else
        upload_file_to_s3 "$source_path" "${dest_dir}"
    fi
}

# Function to upload a single file to S3
upload_file_to_s3() {
    local file="$1"
    local dest_dir="$2"
    local content_type=$(get_content_type "$file")
    aws s3 cp "$file" "s3://${AWS_S3_BUCKET_DOCS}/${dest_dir}/$(basename "$file")" --content-type "$content_type" || {
        echo "Failed to upload $file to S3"
        exit 1
    }
}

# Function to invalidate CloudFront cache
invalidate_cloudfront() {
    local invalidation_path="$1"
    aws cloudfront create-invalidation --distribution-id "${AWS_CLOUDFRONT_DISTRIBUTION_ID_DOCS}" --paths "${invalidation_path}" || {
        echo "Failed to create CloudFront invalidation for ${invalidation_path}"
        exit 1
    }
}

# Main function to publish to site
publish_to_site() {
    local source_path="$1"
    local target_dir="${2:-}"
    local site_dir="docs.powerdns.com"

    local full_target_dir="${site_dir}/${target_dir}"
    upload_to_s3 "$source_path" "$full_target_dir"

    local invalidation_path="/${target_dir}*"
    invalidate_cloudfront "$invalidation_path"

    echo "Published from ${source_path} to docs.powerdns.com${target_dir:+/}${target_dir}"
    echo "Invalidated CloudFront cache for ${invalidation_path}"
}

# Main script execution
if [ "$#" -lt 1 ] || [ "$#" -gt 2 ]; then
    echo "Usage: $0 <SOURCE_PATH> [TARGET_DIR]"
    exit 1
fi

source_path="$1"
target_dir="${2:-}"

publish_to_site "$source_path" "$target_dir"

exit 0