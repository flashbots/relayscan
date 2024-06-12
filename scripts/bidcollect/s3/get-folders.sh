#!/bin/bash
aws --profile r2 s3 ls s3://relayscan-bidarchive/$1 --endpoint-url "https://${CLOUDFLARE_R2_ACCOUNT_ID}.r2.cloudflarestorage.com" | awk '{ print $2 }'