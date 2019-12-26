#!/bin/bash

REPO_ROOT=$(git rev-parse --show-toplevel)

pushd "${REPO_ROOT}"

docker-compose -f docker-compose-local.yml up

popd
