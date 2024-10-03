#!/bin/sh

/app/bin=${/app/bin:-/app/bin}

/app/bin/storagenode dashboard --config-dir /app/config --identity-dir /app/identity $@