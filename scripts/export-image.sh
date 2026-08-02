#!/usr/bin/env bash
set -Eeuo pipefail

engine=${CONTAINER_ENGINE:-docker}
tag=${IMAGE_TAG:-local/vohive-qdc507:0.2.0}
archive=${1:-vohive-qdc507-image.tar}

"$engine" build -t "$tag" .
"$engine" save -o "$archive" "$tag"
echo "Created $archive"
