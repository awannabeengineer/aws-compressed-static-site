#!/bin/bash

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"
SRC_DIR="$SCRIPT_DIR/www"

# compress all supported files
find $SRC_DIR -regextype posix-egrep -regex ".*\.(html|js|css|xml|svg|tff|ico)$" -type f -print0 | xargs -0 gzip -k -9
find $SRC_DIR -regextype posix-egrep -regex ".*\.(html|js|css|xml|svg|tff|ico)$" -type f -print0 | xargs -0 brotli -k -q 11
