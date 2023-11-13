#!/usr/bin/env bash

echo "Compiling cosm-wasm contracts"

OSMOSIS_VERSION=v20.2.1
# For this script to work properly
# We need to copy the contents of the cosmwasm folder of the
# Osmosis repo (https://github.com/osmosis-labs/osmosis/tree/v20.2.1/cosmwasm)
# into the ./tests/nix_tests/cosmwasm folder

git clone -b $OSMOSIS_VERSION --single-branch https://github.com/osmosis-labs/osmosis.git /tmp/osmosis

cp -r /tmp/osmosis/cosmwasm/* ./tests/nix_tests/cosmwasm
rm -rf /tmp/osmosis

cd ./tests/nix_tests/cosmwasm
docker run --rm -v "$(pwd)":/code \
    --mount type=volume,source="$(basename "$(pwd)")_cache",target=/target \
    --mount type=volume,source=registry_cache,target=/usr/local/cargo/registry \
    cosmwasm/workspace-optimizer:0.15.0

# Remove all files and subdirectories except 'artifacts'
# where the compiled contracts are located
find . -mindepth 1 -maxdepth 1 ! -name 'artifacts' -exec rm -r {} \;