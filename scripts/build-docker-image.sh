#!/usr/bin/env bash

set -xeuo pipefail

cd "$(dirname "$(readlink -f "../${BASH_SOURCE[0]}")")"

[ -x bin/podagent ] || (echo "please build the podagent first by running ./build.sh"; exit 1)

. gradle.properties

docker build . -t kaloom/podagent:$version
