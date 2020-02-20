#!/bin/bash

export SCRIPT_LOCT=$( cd $( dirname $0 ); pwd )
cd $SCRIPT_LOCT

cd $SCRIPT_LOCT/MockCwctlSync
mvn package
cd target
MOCK_CWCTL_JAR=`pwd`/`ls MockCwctlSync-*.jar`


echo "Starting Node filewatcher ---------------------------------------------------------"

NODE_LOG=`mktemp`

cd $SCRIPT_LOCT/../Filewatcherd-TypeScript

export CODEWIND_URL_ROOT="http://localhost:9090"
export MOCK_CWCTL_INSTALLER_PATH="$MOCK_CWCTL_JAR"

npm install
npm run serve > $NODE_LOG 2>&1 &
NODE_PID=$!


cd $SCRIPT_LOCT/FilewatcherTests

export DIR=`pwd`

echo "Beginning Node test ----------------------------------------------------------------"

mvn clean test
TEST_ERR_CODE=$?

echo Test complete.

kill $NODE_PID
wait $NODE_PID 2>/dev/null

echo Node Filewatcher log at: $NODE_LOG 

cd $SCRIPT_LOCT

cat $NODE_LOG
#./analyze_log.sh $NODE_LOG

exit $TEST_ERR_CODE
