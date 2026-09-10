#!/bin/sh
/app/Mon.x86_64 -batchmode -nographics -logFile /app/server.log &
export SIDECAR_LOG_PATH=/app/server.log
exec /app/sidecar
