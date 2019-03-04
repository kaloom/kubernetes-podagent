#!/usr/bin/env bash

set -xeuo pipefail

cd "$(dirname "$(readlink -f "../${BASH_SOURCE[0]}")")"

build_dir="$(pwd)/build"
rpm_top_dir=$(mktemp -d -t XXXXXrpmtop)

mkdir -p ${build_dir}

# cleanup on exit
trap "rm -rf ${rpm_top_dir} ${build_dir}" EXIT

[ -x bin/podagent ] || (echo "please build the podagent first by running ./build.sh"; exit 1)
cp rpm/* bin/* ${build_dir}

. gradle.properties

sed -i "s,@@rpm_build_dir@@,${build_dir}," ${build_dir}/podagent.spec

rpmbuild -bb --define "_topdir ${rpm_top_dir}" --define "_pkg_version $version" --define '_pkg_release 1' ${build_dir}/podagent.spec

cp $(find ${rpm_top_dir}/RPMS -name \*.rpm) .
