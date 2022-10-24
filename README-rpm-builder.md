# LightningStream rpm builder

This repository contains the files to create an rpm from https://gitlab.open-xchange.com/powerdns/lightningstream

Please note that currently only packages for centos 7-8-9 (and derivatives) are created

* SUBMODULES

- pdns-builder
- lightningstream
- pdns-linter 

# HOW TO BUILD

To build rpm for centos-7

./builder/build.sh centos-7

To build rpm for centos-8

./builder/build.sh centos-8

To build rpm for centos-9

./builder/build.sh centos-9