#!/usr/bin/env bash

# Package for npm

outDir="npm"
libDir="${outDir}/lib"

set -ex

npm run tslint
rm -rf $libDir $decDir
tsc -d --outDir $outDir --sourceMap false

cp -v package.json package-lock.json $outDir
