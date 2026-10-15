#!/bin/sh
set -eu

: "${SLACK_WEBHOOK_URL:?SLACK_WEBHOOK_URL is required}"

sed "s|\${SLACK_WEBHOOK_URL}|$SLACK_WEBHOOK_URL|" \
  /etc/alertmanager/alertmanager.yml.template \
  > /tmp/alertmanager.yml

exec /bin/alertmanager \
  --config.file=/tmp/alertmanager.yml \
  --storage.path=/alertmanager
