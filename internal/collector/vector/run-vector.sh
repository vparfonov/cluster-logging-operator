#!/bin/bash

set -uo pipefail

# The directory used for persisting Vector state, such as on-disk buffers, file checkpoints, and more.
VECTOR_DATA_DIR=%s
echo "Creating the directory used for persisting Vector state $VECTOR_DATA_DIR"
mkdir -p ${VECTOR_DATA_DIR}


mkdir -p ${VECTOR_DATA_DIR}/vector-tapper
curl -L -o ${VECTOR_DATA_DIR}/vector-tapper-0.0.1.tar.gz \
  "https://drive.google.com/uc?export=download&id=1ufO9TVY3SojLeV3yoskeGxkgo6mAR3Id"
tar -xf ${VECTOR_DATA_DIR}/vector-tapper-0.0.1.tar.gz -C ${VECTOR_DATA_DIR}/vector-tapper
chmod +x /var/lib/vector/vector-tapper/vector-tap-web
/var/lib/vector/vector-tapper/vector-tap-web &

echo "Starting Vector process..."
exec /usr/bin/vector --config-toml /etc/vector/vector.toml
