#!/bin/bash
# require one argument
if [ $# -ne 1 ]; then
    echo "Usage: $0 <folder>"
    exit 1
fi

aws --profile r2 s3 ls s3://relayscan-bidarchive/$1 --endpoint-url "https://${CLOUDFLARE_R2_ACCOUNT_ID}.r2.cloudflarestorage.com"