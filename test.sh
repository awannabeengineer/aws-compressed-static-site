#!/bin/bash

if [ -z "$1" ]; then
  # replace with resulting CloudFront Distribution domain name
  # or pass as argument
  URL=d2w4yo639yatox.cloudfront.net
else
  URL=$1
fi

declare -A encodings
encodings=( ["br"]="Brotli" ["gzip"]="gzip" )

# Test for encoding support
for encoding in "${!encodings[@]}"; do
    response=$(curl -s -I -H "Accept-Encoding: $encoding" "$URL" | grep -i "content-encoding")
    if [[ $response == *"$encoding"* ]]; then
        echo "[✔] ${encodings[$encoding]} support."
    else
        echo "[❌] ${encodings[$encoding]} support."
    fi
done

# Test for brotli preferred over gzip
response_br_gzip=$(curl -s -I -H "Accept-Encoding: gzip, br" "$URL" | grep -i "content-encoding")
if [[ $response_br_gzip == *"br"* ]]; then
  echo "[✔] brotli prefered."
else
  echo "[❌] brotli prefer."
fi
