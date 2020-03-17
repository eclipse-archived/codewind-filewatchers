
@echo off

set CODEWIND_URL_ROOT=http://localhost:9090

rem REPLACE THIS reference to MockCwctlSync JAR, when running via this batch file.
set MOCK_CWCTL_INSTALLER_PATH=c:\Codewind\Git\codewind-filewatchers\Tests\MockCwctlSync\target\MockCwctlSync-0.0.1-SNAPSHOT.jar

npm run watch