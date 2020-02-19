#!/bin/bash

export SCRIPT_LOCT=`dirname $0`
export SCRIPT_LOCT=`cd $SCRIPT_LOCT; pwd`
cd $SCRIPT_LOCT

set -euo pipefail

cd MockCwctlSync
mvn package
cd target
MOCK_CWCTL_JAR=`pwd`/`ls MockCwctlSync-*.jar`

echo $MOCK_CWCTL_JAR
#cd FilewatcherTests
#mvn clean package


