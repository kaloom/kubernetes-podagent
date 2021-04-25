Name:	 podagent
Version: %{_pkg_version}
Release: %{_pkg_release}
License: ASL 2.0
Summary: Kubernetes Podagent

URL: https://kaloom.com

Source0: podagent
Source1: podagent-entrypoint.sh
Source2: podagent.service

%description

%prep
mkdir $RPM_SOURCE_DIR
for f in @@rpm_build_dir@@/*; do cp -a $f $RPM_SOURCE_DIR/; done

%build

%install
mkdir -p $RPM_BUILD_ROOT/opt/kaloom/bin
install -m 744 %{SOURCE0} $RPM_BUILD_ROOT/opt/kaloom/bin/
install -m 744 %{SOURCE1} $RPM_BUILD_ROOT/opt/kaloom/bin/
mkdir -p $RPM_BUILD_ROOT/etc/systemd/system/
install -m 644 %{SOURCE2} $RPM_BUILD_ROOT/etc/systemd/system/

%post
hostname=$(hostname | tr '[A-Z]' ['a-z'])
config_dir=/opt/kaloom/etc
config_file_path=${config_dir}/podagent.conf
log_level=3
log_file=/var/log/cni.log

systemctl daemon-reload

mkdir -p $config_dir
echo "PODAGENT_EXTRA_ARGS='-cni-vendor-name kaloom -logtostderr'" > $config_file_path
echo "PODAGENT_KUBECONFIG=\"${config_dir}/podagent-kubeconfig.yaml\"" >> $config_file_path
echo "PODAGENT_HOSTNAME=\"$hostname\"" >> $config_file_path
echo "export _CNI_LOGGING_LEVEL=${log_level}" >> $config_file_path
echo "export _CNI_LOGGING_FILE=${log_file}" >> $config_file_path

%files
/opt/kaloom/bin/podagent
/opt/kaloom/bin/podagent-entrypoint.sh
/etc/systemd/system/podagent.service
