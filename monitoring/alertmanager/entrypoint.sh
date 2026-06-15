#!/bin/sh
set -e

# Fall back to safe placeholders when the email provider isn't configured,
# so Alertmanager still loads a valid config instead of crash-looping.
# (Alerts won't actually be delivered until real values are set in .env.)
# SMTP provider is configured via .env so it can be swapped (Resend, SendGrid, SES, ...)
# without touching this file or the config template. Defaults point to Resend.
SMTP_SMARTHOST="${SMTP_SMARTHOST:-smtp.resend.com:587}"
SMTP_USERNAME="${SMTP_USERNAME:-resend}"
SMTP_PASSWORD="${SMTP_PASSWORD:-unconfigured}"
ALERT_EMAIL_RECEIVER="${ALERT_EMAIL_RECEIVER:-alerts@localhost}"
ALERT_EMAIL_SENDER="${ALERT_EMAIL_SENDER:-alerts@filmmash.com}"

if [ "${SMTP_PASSWORD}" = "unconfigured" ]; then
  echo "WARNING: SMTP_PASSWORD/ALERT_EMAIL_RECEIVER not set; email alerts are disabled." >&2
fi

# Alertmanager does not expand environment variables in its config file,
# so we render the template into a config with the .env values substituted.
sed \
  -e "s|\${SMTP_SMARTHOST}|${SMTP_SMARTHOST}|g" \
  -e "s|\${SMTP_USERNAME}|${SMTP_USERNAME}|g" \
  -e "s|\${SMTP_PASSWORD}|${SMTP_PASSWORD}|g" \
  -e "s|\${ALERT_EMAIL_RECEIVER}|${ALERT_EMAIL_RECEIVER}|g" \
  -e "s|\${ALERT_EMAIL_SENDER}|${ALERT_EMAIL_SENDER}|g" \
  /etc/alertmanager/alertmanager.tmpl.yml > /tmp/alertmanager.yml

exec /bin/alertmanager --config.file=/tmp/alertmanager.yml "$@"
