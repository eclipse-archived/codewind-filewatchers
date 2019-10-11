#!/usr/bin/env bash

# Package into tarball for consumption by npm dependees

export TSC_OUTDIR="prod"

set -ex

npm ci
npm run tslint
npm run compile-ts-prod
npm prune --production

cp -v package.json package-lock.json $TSC_OUTDIR

version=$(node -e "console.log(require('./package.json').version);")
# version="1.0.0"

tarball=filewatcherd-node_${version}.tar.gz
tar -zcf $tarball $TSC_OUTDIR
# echo $tarball
