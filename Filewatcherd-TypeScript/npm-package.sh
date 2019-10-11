#!/usr/bin/env bash

# Package into tarball for consumption by npm dependees

outDir="npm"

set -ex

npm ci
npm run tslint
npx run tsc -d --outDir $outDir --sourceMap false
npm prune --production

cp -v package.json package-lock.json $outDir

version=$(node -e "console.log(require('./package.json').version);")
# version="1.0.0"

tarball=filewatcherd-node_${version}.tar.gz
tar -zcf $tarball $outDir
# echo $tarball
