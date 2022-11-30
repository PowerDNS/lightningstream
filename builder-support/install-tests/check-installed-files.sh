#!/bin/bash

failed=0

check_file() {
    [ ! -f "$1" ] && echo "FAILED: missing file: $1" && failed=1
}
check_exec() {
    if [ ! -f "$1" ]; then
        echo "FAILED: missing executable file: $1"
        failed=1
    elif [ ! -x "$1" ]; then
        echo "FAILED: file not executable: $1"
        failed=1
    fi
}

check_exec /usr/bin/lightningstream

[ "$failed" = "1" ] && exit 1
exit 0
