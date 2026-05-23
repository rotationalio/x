#!/usr/bin/env bash
# Regenerate plaintext.txt with random hex lines (no realistic secret shapes).
set -euo pipefail
cd "$(dirname "$0")"
out=plaintext.txt
: >"$out"
# Byte lengths to hex-encode (openssl rand -hex N emits 2*N hex chars).
for nbytes in 8 16 24 32 48 64 96 128 192 256 384 512 768 1024 1536 2048; do
  for _ in 1 2 3; do
    openssl rand -hex "$nbytes" >>"$out"
  done
done
wc -l "$out"
