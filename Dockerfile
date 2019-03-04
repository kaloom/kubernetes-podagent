FROM centos:centos7

RUN install -m 755 -d /opt/kaloom/etc && \
    install -m 755 -d /opt/kaloom/bin && \
    yum install -y epel-release && yum install -y jq && \
    yum clean all

COPY bin/podagent scripts/podagent-entrypoint.sh /opt/kaloom/bin/

ENTRYPOINT ["/opt/kaloom/bin/podagent-entrypoint.sh"]
