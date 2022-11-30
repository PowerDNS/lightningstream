#!/bin/sh

set -ex

/usr/bin/lightningstream --help | grep --silent instance

