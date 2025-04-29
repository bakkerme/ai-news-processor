#!/bin/bash

echo "Initialising AI News Processor"

if [ "$ANP_DEBUG_SKIP_CRON" = "true" ]; then
    echo "Debug mode: Skipping cron setup and running main directly"
    /app/main
else
    sed -i "s/\[INSERT\]/$ANP_CRON_SCHEDULE/" /etc/cron.d/appcron
    printenv  >> /etc/environment
    
    cat /etc/cron.d/appcron | crontab -
    
    echo "Running at $ANP_CRON_SCHEDULE"
    crond -f -L "/dev/stdout"
fi
