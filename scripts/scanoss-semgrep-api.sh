#!/bin/bash

##########################################
#
# This script is designed to run by Systemd SCANOSS Semgrep API service.
# It rotates scanoss log file and starts SCANOSS Semgrep API.
# Install it in /usr/local/bin
#
################################################################
DEFAULT_ENV="prod"
ENVIRONMENT="${1:-$DEFAULT_ENV}"
LOGFILE=/var/log/scanoss/semgrep/scanoss-semgrep-api-$ENVIRONMENT.log
CONF_FILE=/usr/local/etc/scanoss/semgrep/app-config-${ENVIRONMENT}.json

# Rotate log
if [ -f "$LOGFILE" ] ; then
  echo "rotating logfile..."
  TIMESTAMP=$(date '+%Y%m%d-%H%M%S')
  BACKUP_FILE=$LOGFILE.$TIMESTAMP
  cp "$LOGFILE" "$BACKUP_FILE"
  gzip -f "$BACKUP_FILE"
fi
echo > "$LOGFILE"

echo > $LOGFILE
# Start scanoss-semgrep-api
echo "starting SCANOSS Semgrep API"
exec /usr/local/bin/scanoss-semgrep-api --json-config "$CONF_FILE" > "$LOGFILE" 2>&1