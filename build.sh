#!/usr/bin/env bash

rm -fr dist/

mkdir -p dist

echo "Copying default profile..."
# cp -v *.profile dist/
cp -v default.profile.sample dist/default.profile

echo "Copying default config..."
cp -v fox.config.sample dist/fox.config

echo "Building application..."
go build -o dist/fox fox.go
