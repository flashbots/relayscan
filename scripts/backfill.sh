#!/bin/bash
set -e
dir=$( dirname -- "$0"; )
cd $dir
cd ..
source .env.prod
/usr/local/go/bin/go run . data-api-backfill 2>&1 | /usr/bin/tee /var/log/relayscan.log