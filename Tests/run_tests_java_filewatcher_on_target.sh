#!/bin/bash

export SCRIPT_LOCT=$( cd $( dirname $0 ); pwd )
cd $SCRIPT_LOCT

if [ -z "$1" ]; then
	echo "Error: first parameter should be root directory of codewind-eclipse repository"
	exit 1
fi
CODEWIND_ECLIPSE_ROOT=$1



cd $SCRIPT_LOCT/MockCwctlSync
mvn package
cd target
MOCK_CWCTL_JAR=`pwd`/`ls MockCwctlSync-*.jar`


echo "Starting Java filewatcher ---------------------------------------------------------"

JAVA_LOG=`mktemp`

cd $CODEWIND_ECLIPSE_ROOT/dev/org.eclipse.codewind.filewatchers.core
mvn clean install

cd $CODEWIND_ECLIPSE_ROOT/dev/org.eclipse.codewind.filewatchers.standalonenio
mvn clean package

export CODEWIND_URL_ROOT="http://localhost:9090"
export MOCK_CWCTL_INSTALLER_PATH="$MOCK_CWCTL_JAR"

java -jar target/org.eclipse.codewind.filewatchers.standalonenio-*.jar > $JAVA_LOG 2>&1 &
JAVA_PID=$!


cd $SCRIPT_LOCT/FilewatcherTests

export DIR=`pwd`

echo "Beginning Java test ----------------------------------------------------------------"

mvn clean test
TEST_ERR_CODE=$?

echo Test complete.

kill $JAVA_PID
wait $JAVA_PID 2>/dev/null

echo Node Filewatcher log at: $JAVA_LOG 

cd $SCRIPT_LOCT

cat $JAVA_LOG
#./analyze_log.sh $JAVA_LOG

exit $TEST_ERR_CODE

