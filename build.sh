#!/bin/bash

COMMIT_HASH=$(git rev-parse --short HEAD)
BRANCH_NAME=$(git branch --show-current)

PACKAGE_PATH="github.com/govdbot/govd/bot/handlers"

echo "Building with commit hash: ${COMMIT_HASH}"
echo "Branch name: ${BRANCH_NAME}"
go build -ldflags="-X '${PACKAGE_PATH}.buildHash=${COMMIT_HASH}' -X '${PACKAGE_PATH}.branchName=${BRANCH_NAME}'"

if [ $? -eq 0 ]; then
    echo "Build completed successfully"
else
    echo "Build failed"
    exit 1
fi