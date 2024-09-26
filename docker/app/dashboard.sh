#!/bin/sh

BINARY_DIR=${BINARY_DIR:-/app/bin}

${BINARY_DIR}/storagenode dashboard --config-dir /app/config --identity-dir /app/identity $@