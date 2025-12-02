#!/bin/bash

# Dstore Documentation Building Script
#
# This script makes the lightningstream HTML documentation ready for copying to S3
#
# Environment Variables Required:
# - AWS_ACCESS_KEY_ID: The AWS access key ID
# - AWS_SECRET_ACCESS_KEY: The AWS secret access key
# - AWS_REGION: The AWS region where resources are located
# - AWS_S3_BUCKET_DOCS: The name of the S3 bucket for documentation
# - BUILD_PATH: The root of the lightningstream directory
#
# Usage:
# ./mkdocs.sh <MKDOCS_YAML> <VERSION> <MKDOCS_IMAGE>

set -e  # Exit immediately if a command exits with a non-zero status

# Main script execution
if [ "$#" -lt 1 ] || [ "$#" -gt 3 ]; then
    echo "Usage: $0 <MKDOCS_YAML> <VERSION> <MKDOCS_IMAGE>"
    exit 1
fi

mkdocs_file="$1"
version="$2"
image="$3"

publish_script="${BUILD_PATH}/.github/scripts/publish_to_s3.sh"

# Prep temporary output location
mkdir -p ${PWD}/output/"${version}"

docker run -v "${PWD}:${PWD}" $image sh -c "pip install mkdocs-swagger-ui-tag && mkdocs  build -f $mkdocs_file -d ${PWD}/output/${version}"

latestVersion=`aws s3 ls s3://"${AWS_S3_BUCKET_DOCS}"/docs.powerdns.com/lightningstream/ | awk '{print $2}' | grep -v latest | awk -F '/' '/\// {print $1}' | sort -V | tail -1`

if [ "$latestVersion" == "" ]; then
  latestVersion="0"
fi

echo "Publishing version $version. Latest version already in S3 is $latestVersion"

$publish_script ${PWD}/output/${version} lightningstream/${version}

if (( $(echo "$latestVersion" "$version" | awk '{if ($1 < $2) print 1;}') )); then
  echo "This version is newer than the latest version in S3, publishing this version to latest"
  $publish_script ${PWD}/output/${version} lightningstream/latest
  latestVersion="$version"
fi

# Build versions.json
versionsData=$(echo "[]" | jq)

while read -r docsVersion; do
  if [ "$docsVersion" != "" ] && [ "$docsVersion" != "latest" ]; then
    if [ $docsVersion == $latestVersion ]; then
      versionsData=$(echo $versionsData | jq ". += [{\"title\": \"${docsVersion}\", \"version\": \"${latestVersion}\", \"aliases\": [\"latest\"]}]")
    else
      versionsData=$(echo $versionsData | jq ". += [{\"title\": \"${docsVersion}\", \"version\": \"${docsVersion}\", \"aliases\": []}]")
    fi
  fi
done < <(aws s3 ls s3://"${AWS_S3_BUCKET_DOCS}"/docs.powerdns.com/lightningstream/ | awk '{print $2}' | awk -F '/' '/\// {print $1}')

echo ${versionsData} > ${PWD}/output/versions.json

$publish_script ${PWD}/output/versions.json lightningstream

$publish_script ${PWD}/doc/html/index.html lightningstream

exit 0
