#!/bin/bash
set -e
dir=$( dirname -- "$0"; )
cd $dir
cd ..
source .env.prod
/server/relayscan/relayscan data-api-backfill 2>&1 | /usr/bin/tee /var/log/relayscan.log
/server/relayscan/relayscan check-payload-value 2>&1 | /usr/bin/tee -a /var/log/relayscan.log