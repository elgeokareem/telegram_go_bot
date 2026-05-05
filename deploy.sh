#!/bin/bash

set -e

echo "Changing to the repository directory..."
cd ~/documents/projects/telegram_go_bot/ || exit 1

export PATH="$PATH:/usr/local/go/bin"

echo "Pulling latest changes..."
git pull origin master

echo "Running migrations..."
make migrate-up

echo "Building the app..."
make build
make build-worker

echo "Stopping the services..."
systemctl stop go-bot-telegram.service
systemctl stop event-reminder-worker.service 2>/dev/null || true

echo "Copying the build"
cp ~/documents/projects/telegram_go_bot/telegram_go_bot /opt/telegram_go_bot/
cp ~/documents/projects/telegram_go_bot/event_reminder_worker /opt/telegram_go_bot/
cp ~/documents/projects/telegram_go_bot/.env /opt/telegram_go_bot/

echo "Restarting the apps..."
chmod +x /opt/telegram_go_bot/telegram_go_bot
chmod +x /opt/telegram_go_bot/event_reminder_worker
systemctl daemon-reload
systemctl restart go-bot-telegram.service
systemctl enable event-reminder-worker.service 2>/dev/null || true
systemctl restart event-reminder-worker.service

echo "Deployment complete!"
