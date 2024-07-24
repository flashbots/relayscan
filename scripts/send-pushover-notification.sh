#!/bin/bash
if [ -z "$1" ]; then
  echo "Usage: $0 <message>"
  exit 1
fi

curl -s \
        --form-string "token=$PUSHOVER_APP_TOKEN" \
        --form-string "user=$PUSHOVER_APP_KEY" \
        --form-string "message=$1" \
        https://api.pushover.net/1/messages.json
