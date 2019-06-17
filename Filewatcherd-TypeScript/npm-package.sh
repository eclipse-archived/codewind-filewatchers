#!/usr/bin/env bash

# Package for npm

outDir="npm"
libDir="${outDir}/lib"

set -ex

npm run tslint
rm -rf $libDir
npm ci
tsc -d --outDir $outDir --sourceMap false

cp -v package.json package-lock.json $outDir

version=$(node -e "console.log(require('./package.json').version);")
# version="1.0.0"

tarball=filewatcherd-node_${version}.tar.gz
tar -zcf $tarball $outDir
# echo $tarball
