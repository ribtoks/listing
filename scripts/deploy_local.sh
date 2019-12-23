#!/bin/bash

REPO_ROOT=$(git rev-parse --show-toplevel)
LAMBDA_DIR="${REPO_ROOT}"

pushd $LAMBDA_DIR

serverless deploy --stage local --region "us-east-1" --verbose

popd
