#
# spec file for package pdns-prometheus-webhook-snmp
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


Name:           %{appname}
Version:        %{getenv:BUILDER_VERSION}
Release:        0
Summary:        PowerDNS LightningStream
License:        Proprietary
Group:          System/Management
Url:            https://gitlab.open-xchange.com/powerdns/lightningstream
Source0:        %{name}-%{version}.tar.gz
BuildArch:      x86_64

BuildRequires:  epel-rpm-macros

# Requires:       epel-release

%description
LMDB to S3 bucket syncer for the PowerDNS Auth LMDB backend

%prep
%setup -q -n %{appname}-%{version}

%build
./build.sh

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

%post

%preun
%systemd_preun %{appname}.service

%postun
%systemd_postun %{appname}.service

%clean
rm -rf %{buildroot}

%files
%{_docdir}/README.md
%{_docdir}/%{name}.example.yaml
%{_bindir}/%{appname}
%{_unitdir}/%{appname}.service
%{confdir}/%{appname}.yaml