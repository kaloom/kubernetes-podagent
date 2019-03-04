#!/usr/bin/env bash

set -eo pipefail

running_as_pod=0
cninet_top_dir=
cni_cfg_file=

# check if we're running as a Pod in kuberentes
if [ -h /var/run/secrets/kubernetes.io/serviceaccount/token ]; then
    running_as_pod=1
    cninet_top_dir=/host
fi

while :; do
    for f in ${cninet_top_dir}/etc/cni/net.d/*; do
	if [ -f $f ]; then # wait until we see the 1st lexically cni config file
	    echo "cni config $f is for the system cni-plugin in effect"
	    cni_cfg_file=$f
	    break 2
	fi
    done
    echo "waiting for a cni config file"
    sleep 10
done

cnitype=$(jq -r .type < $cni_cfg_file)
if [ "$cnitype" != "kactus" ]; then
    echo "system is configured with an unsupported $cnitype cni-plugin"
    echo "currently only kactus know how to work with dynamic network attachment"
    exit 1
fi

if [ -e /opt/kaloom/etc/podagent.conf ]; then
    source /opt/kaloom/etc/podagent.conf
fi

if [ $running_as_pod -eq 1 ]; then
    # create /etc/cni/net.d inside the container and copy the host cni
    # config file as is stripping from it kubeconfig element to use for
    # in-cluster authentication
    install -m 755 -d /etc/cni/net.d
    cat $cni_cfg_file | jq . | sed '/kubeconfig/d' > /etc/cni/net.d/00-kactus.conf
else
    while [ -n "$PODAGENT_KUBECONFIG" ] && [ ! -r "$PODAGENT_KUBECONFIG" ]; do
	echo "waiting for $kubecfg_file"
	sleep 10
    done
fi

if [ -f "$PODAGENT_KUBECONFIG" ]; then
    kubeconfig_args="-kubeconfig $PODAGENT_KUBECONFIG"
fi

/opt/kaloom/bin/podagent -node $PODAGENT_HOSTNAME $kubeconfig_args $PODAGENT_EXTRA_ARGS
