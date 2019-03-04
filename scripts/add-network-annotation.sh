#!/bin/sh

set -e

pod=$1
net=$2
ns=${3:-"default"}

if [ -z "$pod" ] || [ -z "$net" ]; then
    echo "$0 <pod> <net> <namespace>"
    exit 1
fi

current_nets=$(kubectl get pod $pod -o jsonpath='{.metadata.annotations.networks}' | jq .[] | sed -e '$s/}/},/')

kubectl annotate -n ${ns} --overwrite pods ${pod} networks="[ $current_nets { \"name\": \"${net}\"} ]"
