#!/bin/bash
sed -i "s/\[INSERT\]/$ANP_CRON_SCHEDULE/" /etc/cron.d/appcron
printenv  >> /etc/environment

cat /etc/cron.d/appcron | crontab -

echo "Initialising AI News Processor"
echo "Running at $ANP_CRON_SCHEDULE"
crond -f -L "/dev/stdout"
