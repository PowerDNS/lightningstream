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

<<<<<<< HEAD
#check_exec /usr/bin/demo-a
#check_exec /usr/bin/demo-b
=======
check_exec /usr/bin/lightningstream
>>>>>>> 7190c502c5f726c0c43e3710b2aeaf72d383239d

[ "$failed" = "1" ] && exit 1
exit 0
