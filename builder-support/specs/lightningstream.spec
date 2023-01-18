#
# Copyright (c) 2019-2020 SUSE LLC
#
# All modifications and additions to the file contributed by third parties
# remain the property of their copyright owners, unless otherwise agreed
# upon. The license for this file, and modifications and additions to the
# file, is the same license as for the pristine package itself (unless the
# license for the pristine package is not an Open Source License, in which
# case the license is the MIT License). An "Open Source License" is a
# license that conforms to the Open Source Definition (Version 1.9)
# published by the Open Source Initiative.

# Please submit bugfixes or comments via http://bugs.opensuse.org/

%define  debug_package %{nil}
%define appname lightningstream
%define _bindir /usr/bin/
%define _docdir /usr/share/doc/%{appname}
%define confdir /etc/%{appname}
%define changelog CHANGELOG.md
%define readme README.md
%define src_version %{getenv:BUILDER_VERSION}

# FIXME: decide on user, depends on LMDB
%global installuser lightningstream
%global installgroup %{installuser}


Name:           %{appname}
Version:        %{getenv:BUILDER_RPM_VERSION}
Release:        %{getenv:BUILDER_RPM_RELEASE}%{dist}
Summary:        PowerDNS LightningStream
License:        Proprietary
Group:          System/Management
Url:            https://powerdns.com/
Source0:        %{name}-%{src_version}.tar.gz
#BuildArch:      x86_64
Requires(pre):  shadow-utils
%systemd_requires

BuildRequires:  epel-rpm-macros

# Requires:       epel-release

%description
LMDB to S3 bucket syncer

%prep
%setup -q -n %{appname}-%{src_version}

%build
./build.sh
# ./test.sh

%install
mkdir -p %{buildroot}%{_bindir}/
cp bin/%{appname} %{buildroot}%{_bindir}%{appname}
mkdir -p %{buildroot}%{_docdir}
cp %{readme} %{buildroot}%{_docdir}/%{readme}
cp files/%{appname}.yaml %{buildroot}%{_docdir}/%{appname}.example.yaml
mkdir -p %{buildroot}/usr/lib/systemd/system/
mkdir -p %{buildroot}/%{confdir}/
cp files/%{appname}.service %{buildroot}/%{_unitdir}/%{appname}.service
cp files/%{appname}.yaml %{buildroot}%{confdir}/%{appname}.yaml
sed -i 's|_BINARY_|%{_bindir}%{appname}|g' %{buildroot}/%{_unitdir}/%{appname}.service
sed -i 's|_CONFIG_|%{confdir}/%{appname}.yaml|g' %{buildroot}/%{_unitdir}/%{appname}.service

%pre
getent group %{installgroup} > /dev/null || groupadd -r %{installgroup}
getent passwd %{installuser} > /dev/null || useradd -r -g %{installgroup} -d / -s /sbin/nologin -c "%{installuser} user" %{installuser}
exit 0

%post
# FIXNE: This needs to become a template service with support for multiple instances
%systemd_post %{appname}.service

%preun
%systemd_preun %{appname}.service

%postun
%systemd_postun_with_restart %{appname}.service

%clean
rm -rf %{buildroot}

%files
%{_docdir}/README.md
%{_docdir}/%{name}.example.yaml
%{_bindir}/%{appname}
%{_unitdir}/%{appname}.service
%{confdir}/%{appname}.yaml
