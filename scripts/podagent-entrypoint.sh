#!/bin/sh

while :; do
    for f in /etc/cni/net.d/*; do
	if [ -r $f ]; then # wait until we see the 1st readable cni config
	    echo "cni config $f is readable"
	    break 2
	fi
    done
    echo "waiting for a cni config file"
    sleep 10
done

if [ -e /opt/kaloom/etc/podagent.conf ]; then
    source /opt/kaloom/etc/podagent.conf
fi

while [ -n "$PODAGENT_KUBECONFIG" ] && [ ! -r "$PODAGENT_KUBECONFIG" ]; do
    echo "waiting for $kubecfg_file"
    sleep 10
done

if [ -f "$PODAGENT_KUBECONFIG" ]; then
    kubeconfig_args="-kubeconfig $PODAGENT_KUBECONFIG"
fi

/opt/kaloom/bin/podagent -node $PODAGENT_HOSTNAME $kubeconfig_args $PODAGENT_EXTRA_ARGS
