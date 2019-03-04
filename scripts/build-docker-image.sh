#!/usr/bin/env bash

set -euo pipefail

cd "$(dirname "$(readlink -f "../${BASH_SOURCE[0]}")")"

[ -x bin/podagent ] || (echo "please build the podagent first by running ./build.sh"; exit 1)

. gradle.properties

docker build . -t kaloom/podagent:$version
docker tag kaloom/podagent:$version kaloom/podagent:latest
