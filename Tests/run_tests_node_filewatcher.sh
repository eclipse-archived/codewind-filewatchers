#!/bin/bash


export SCRIPT_LOCT=$( cd $( dirname $0 ); pwd )
cd $SCRIPT_LOCT



# cd ~

# Install nvm to easily set version of node to use
# curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.34.0/install.sh | bash
# export NVM_DIR="$HOME/.nvm" 
# set -a
# echo pre1
# . $NVM_DIR/nvm.sh
# echo pre2
# npm config delete prefix
# nvm i 10

# echo post





cd $SCRIPT_LOCT/MockCwctlSync
mvn package
cd target
MOCK_CWCTL_JAR=`pwd`/`ls MockCwctlSync-*.jar`


echo "Starting Node filewatcher ---------------------------------------------------------"

NODE_LOG=`mktemp`

cd $SCRIPT_LOCT/../Filewatcherd-TypeScript

export CODEWIND_URL_ROOT="http://localhost:9090"
export MOCK_CWCTL_INSTALLER_PATH="$MOCK_CWCTL_JAR"

npm ci
npm run compile-ts
npm run serve > $NODE_LOG 2>&1 &
NODE_PID=$!

echo NODE_PID Is $NODE_PID

cd $SCRIPT_LOCT/FilewatcherTests

export DIR=`pwd`

echo "Beginning Node test ----------------------------------------------------------------"

mvn clean test
TEST_ERR_CODE=$?

echo Test complete.

kill $NODE_PID
wait $NODE_PID 2>/dev/null
# killall node

echo Node Filewatcher log at: $NODE_LOG 

cd $SCRIPT_LOCT

cat $NODE_LOG
#./analyze_log.sh $NODE_LOG

exit $TEST_ERR_CODE

