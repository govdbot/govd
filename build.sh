#!/bin/bash

COMMIT_HASH=$(git rev-parse --short HEAD)
BRANCH_NAME=$(git branch --show-current)

PACKAGE_PATH="main"

echo "building with commit hash: ${COMMIT_HASH}"
echo "branch name: ${BRANCH_NAME}"

go build -ldflags="-X '${PACKAGE_PATH}.buildHash=${COMMIT_HASH}' -X '${PACKAGE_PATH}.branchName=${BRANCH_NAME}'" -o govd ./cmd/main.go

if [ $? -eq 0 ]; then
    echo "build completed successfully"
else
    echo "build failed"
    exit 1
fi