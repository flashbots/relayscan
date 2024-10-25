#!/bin/bash
set -e
dir=$( dirname -- "$0"; )
cd $dir
cd ..
source .env.prod
./relayscan core bid-adjustments-backfill 2>&1 | /usr/bin/tee -a /var/log/relayscan.log
