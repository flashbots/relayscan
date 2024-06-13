#!/bin/bash
#
# Check health of relayscan.io and send notifications if state changes.
#
# https://www.relayscan.io/healthz
#
# This script is intended to be run as a cron job.
#
# It uses a temporary file to not send multiple notifications and store the error.
#
set -o errexit
set -o nounset
set -o pipefail

url="https://www.relayscan.io/healthz"
# url="localhost:9060/healthz"
check_fn="/tmp/relayscan-error.txt"
check_cmd="curl -s $url"

# load environment variables $PUSHOVER_APP_TOKEN and $PUSHOVER_APP_KEY
source "$(dirname "$0")/../.env.prod"

function send_notification() {
        curl -s \
                --form-string "token=$PUSHOVER_APP_TOKEN" \
                --form-string "user=$PUSHOVER_APP_KEY" \
                --form-string "message=$1" \
                https://api.pushover.net/1/messages.json
}

function error() {
    # don't run if notification was alreaty sent
    if [ -f $check_fn ]; then
        return
    fi

    echo "relayscan.io is unhealthy"
    send_notification "relayscan.io is unhealthy"
    curl -vvvv $url > $check_fn 2>&1
}

function reset() {
    # Don't run if there is no error
    if [ ! -f $check_fn ]; then
        return
    fi

    rm $check_fn
    echo "relayscan.io is healthy again"
    send_notification "relayscan.io is healthy again"
}

# Allow errors, to catch curl error exit code
set +e
# echo $check_cmd
$check_cmd
if [ $? -eq 0 ]; then
    echo "All good"
    reset
else
    echo "curl error $?"
    error
fi
