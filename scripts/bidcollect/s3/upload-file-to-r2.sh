#!/bin/bash
src=$1
target=$2
if [ -z "$src" ]; then
    echo "Usage: $0 <local_file> [<s3_target>"]
    exit 1
fi

# auto-fill target if not given
if [ -z "$target" ]; then
    # remove "/mnt/data/relayscan-bidarchive/" prefix from src and make it the S3 prefix
    target="/ethereum/mainnet/${src#"/mnt/data/relayscan-bidarchive/"}"
fi

echo "uploading $src to S3 $target ..."
aws --profile r2 s3 cp $src s3://relayscan-bidarchive$target --endpoint-url "https://${CLOUDFLARE_R2_ACCOUNT_ID}.r2.cloudflarestorage.com"
