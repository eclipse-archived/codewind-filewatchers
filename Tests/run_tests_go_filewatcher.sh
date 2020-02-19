#!/bin/bash

export SCRIPT_LOCT=$( cd $( dirname $0 ); pwd )
cd $SCRIPT_LOCT


cd $SCRIPT_LOCT/MockCwctlSync
mvn package
cd target
MOCK_CWCTL_JAR=`pwd`/`ls MockCwctlSync-*.jar`


echo "Starting Go filewatcher ---------------------------------------------------------"

GO_LOG=`mktemp`

cd ..
pushd . > /dev/null

cd Filewatcherd-Go/src/codewind
go build -race 

export CODEWIND_URL_ROOT="http://localhost:9090"
export MOCK_CWCTL_INSTALLER_PATH="$MOCK_CWCTL_JAR"

./codewind > $GO_LOG 2>&1 &
GO_PID=$!

popd > /dev/null

cd Tests/FilewatcherTests

export DIR=`pwd`

echo "Beginning GO test ----------------------------------------------------------------"

mvn clean test
TEST_ERR_CODE=$?

echo Test complete.

kill $GO_PID
wait $GO_PID 2>/dev/null

echo GO Filewatcher log at: $GO_LOG 

cd $SCRIPT_LOCT

#./analyze_log.sh $GO_LOG

exit $TEST_ERR_CODE

