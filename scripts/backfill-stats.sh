#!/bin/bash
set -e
dir=$( dirname -- "$0"; )
cd $dir
cd ..
source .env.prod
./relayscan core update-builder-stats --backfill --daily --verbose 2>&1 | /usr/bin/tee -a /var/log/relayscan-stats.log
