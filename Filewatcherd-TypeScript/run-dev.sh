#!/bin/bash

export CODEWIND_URL_ROOT=http://localhost:9090

export MOCK_CWCTL_INSTALLER_PATH=`cd ../;pwd`/Tests/MockCwctlSync/target/MockCwctlSync-0.0.1-SNAPSHOT.jar

npm run watch
