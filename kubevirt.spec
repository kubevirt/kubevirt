#debuginfo not supported with Go
%global debug_package %{nil}

%global package_name kubevirt
%{!?commit: %global commit %(git rev-parse HEAD)}
%{!?build_hash: %global build_hash %(c=%{commit}; echo ${c:0:7})}
%{!?spec_release: %global spec_release 1}
%{!?version: %global version 0.7.0}
%{!?release: %global release %{spec_release}.%{build_hash}}
%{!?kubevirt_git_version: %global kubevirt_git_version v%{version}-%{release}}
%{!?docker_prefix: %global docker_prefix docker.io/kubevirt}
%{!?libvirt_version: %global libvirt_version 4.2.0}
%global container_release_tag %{version}
%global sudo_version 1.8.0
%global pkg_namespace kube-system

Name:           %{package_name}
Version:        %{version}
Release:        %{release}
Summary:        %{package_name} - virtual machine management add-on for Kubernetes

License:        ASL 2.0
URL:            %(git config --get remote.origin.url)
Source0:        %{name}-%{version}.tar.gz

BuildRequires:  libvirt-devel >= %{libvirt_version}

%description
kubevirt distribution

%package	    virt-api
Summary:        %{package_name} - virt-api 
%description virt-api
kubevirt virt-api

%package	    virt-controller
Summary:        %{package_name} - virt-controller
%description virt-controller
kubevirt virt-controller

%package	    virt-handler
Summary:        %{package_name} - virt-handler
%description virt-handler
kubevirt virt-handler

%package	    virt-launcher
Summary:        %{package_name} - virt-launcher
Requires:       libvirt >= %{libvirt_version}
Requires:       sudo >= %{sudo_version}
%description virt-launcher
kubevirt virt-launcher

%package        virtctl
Summary:        %{package_name} - virtctl
%description virtctl
kubevirt virtctl

%package        virtctl-redistributable
Summary:        %{package_name} - virtctl-redistributable
%description virtctl-redistributable
kubevirt virtctl executables for linux, MACos, Windows

%package	    manifests
Summary:        %{package_name} - manifests
BuildArch:      noarch
%description manifests
kubevirt manifests

%prep
%setup -q -n %{name}-%{commit}

%build

function j2() {
    export namespace=%{pkg_namespace}
    export docker_prefix=%{docker_prefix}
    export docker_tag=%{container_release_tag}

    cat $@ | perl -p -e 's/\{\{ ([^}]+) \}\}/defined $ENV{$1} ? $ENV{$1} : $&/eg'
}

typeset -fx j2

mkdir -p go/src/kubevirt.io go/pkg
ln -s ../../../ go/src/kubevirt.io/kubevirt
export GOPATH=$(pwd)/go
cd ${GOPATH}/src/kubevirt.io/kubevirt
KUBEVIRT_GO_BASE_PKGDIR="${GOPATH}/pkg" KUBEVIRT_VERSION=%{version} KUBEVIRT_SOURCE_DATE_EPOCH="$(date +%s)" KUBEVIRT_GIT_COMMIT=%{commit} KUBEVIRT_GIT_VERSION=%{kubevirt_git_version} KUBEVIRT_GIT_TREE_STATE="clean" ./hack/build-go.sh install cmd/virt-controller cmd/virt-launcher cmd/virt-handler cmd/virtctl cmd/virt-api
DOCKER_PREFIX=%{docker_prefix} DOCKER_TAG=%{container_release_tag} ./hack/build-manifests.sh 
./hack/build-copy-artifacts.sh

%install
mkdir -p %{buildroot}%{_bindir}

install -d -m 0755 %{buildroot}%{_datadir}/%{name}/virt-launcher
install -p -m 0755 _out/cmd/virt-launcher/sock-connector %{buildroot}%{_datadir}/%{name}/virt-launcher/
install -p -m 0777 _out/cmd/virt-launcher/libvirtd.sh %{buildroot}%{_datadir}/%{name}/virt-launcher/
install -p -m 0755 _out/cmd/virt-launcher/entrypoint.sh %{buildroot}%{_datadir}/%{name}/virt-launcher/
install -d -m 0755 %{buildroot}/%{_sysconfdir}/sudoers.d/
install -p -m 0640 _out/cmd/virt-launcher/%{name}-sudo %{buildroot}/%{_sysconfdir}/sudoers.d/%{name}

install -p -m 0755 _out/cmd/virt-controller/virt-controller %{buildroot}%{_bindir}/
install -p -m 0755 _out/cmd/virt-api/virt-api %{buildroot}%{_bindir}/
install -p -m 0755 _out/cmd/virt-handler/virt-handler %{buildroot}%{_bindir}/
install -p -m 0755 _out/cmd/virt-launcher/virt-launcher %{buildroot}%{_bindir}/
install -p -m 0755 _out/cmd/virtctl/virtctl %{buildroot}%{_bindir}/
install -d -m 0755 %{buildroot}%{_datadir}/%{name}/linux
install -p -m 0755 _out/cmd/virtctl/virtctl-%{version}-linux-amd64 %{buildroot}%{_datadir}/%{name}/linux/virtctl
install -d -m 0755 %{buildroot}%{_datadir}/%{name}/macosx
install -p -m 0755 _out/cmd/virtctl/virtctl-%{version}-darwin-amd64 %{buildroot}%{_datadir}/%{name}/macosx/virtclt
install -d -m 0755 %{buildroot}%{_datadir}/%{name}/windows
install -p -m 0755 _out/cmd/virtctl/virtctl-%{version}-windows-amd64.exe %{buildroot}%{_datadir}/%{name}/windows/virtctl.exe

install -d -m 0755 %{buildroot}%{_datadir}/%{name}/manifests
install -d -m 0755 %{buildroot}%{_datadir}/%{name}/templates
cp -r _out/manifests/* %{buildroot}%{_datadir}/%{name}/manifests/
cp -r _out/templates/* %{buildroot}%{_datadir}/%{name}/templates/

install -p -m 0755 cluster/examples/vm-template-rhel7.yaml %{buildroot}%{_datadir}/%{name}/manifests/vm-template-rhel7.yaml
install -p -m 0755 cluster/examples/vm-template-fedora.yaml %{buildroot}%{_datadir}/%{name}/manifests/vm-template-fedora.yaml
install -p -m 0755 cluster/examples/vm-template-windows2012r2.yaml %{buildroot}%{_datadir}/%{name}/manifests/vm-template-windows2012r2.yaml


# %files
# %doc

%files virt-controller
%{_bindir}/virt-controller

%files virt-api
%{_bindir}/virt-api

%files virt-handler
%{_bindir}/virt-handler

%files virt-launcher
%{_bindir}/virt-launcher
%{_datadir}/%{name}/virt-launcher/sock-connector
%{_sysconfdir}/sudoers.d/%{name}
%{_datadir}/%{name}/virt-launcher/libvirtd.sh
%{_datadir}/%{name}/virt-launcher/entrypoint.sh

%files virtctl
%{_bindir}/virtctl

%files manifests
%{_datadir}/%{name}/manifests/
%{_datadir}/%{name}/templates/

%files virtctl-redistributable
%{_datadir}/%{name}/linux/
%{_datadir}/%{name}/macosx/
%{_datadir}/%{name}/windows/

%changelog
* Tue Jul 24 2018 - Tommy Hughes <tchughesiv@gmail.com>
- rpm spec file, build script, & travis test mods
