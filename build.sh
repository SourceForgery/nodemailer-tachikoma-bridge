#!/bin/bash -eu

#rm -rf build/

mkdir -p build/protobuf/
mkdir -p build/generated/
#(
#  cd build/protobuf
#  wget -q https://github.com/SourceForgery/tachikoma/releases/download/1.0.166/tachikoma-frontend-api-proto-1.0.166.zip -O api.zip
#  unzip api.zip > /dev/null
#)

./gow generate
./gow build
