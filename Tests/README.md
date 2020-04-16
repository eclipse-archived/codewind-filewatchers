## Introduction to Filewatcher Tests

Projects:
- *FileWatcherTests*: This project contains the actual tests themselves, plus a test case framework which mocks the Filewatcher API of the Codewind server.
- *MockCwctlSync*: A mock version of the `cwctl project sync` command, which performs the same general task as that command: the filewatcher calls this whenever file changes are detected, and it is up to MockCwctlSync to detect and report back those changes to the  test case framework.

There is a single test suite in `org.eclipse.codewind.filewatchers.test.tests.FilewatcherTests`, which contains a set of integrations tests which can be used to test any filewatcher (Go/Java/Node). These tests do not require a Codewind server to be running, nor to have Codewind installed. These tests fully test all features of the filewatcher (the only things not tested error handling/debug code that are not used outside of development/error conditions)

## Running the tests

You can see the root `Jenkinsfile` of this repo for the steps that our CI uses to install and run the tests. The steps I've listed below are the manual version of those CI steps.

#### Prerequisites:
- Ensure that Maven is installed and on your path
- Ensure that a Java JDK is installed and on your path.
- If testing the Go filewatcher, ensure that Go is installed and on your path.
- If testing the Node filewatcher, ensure that Node and NPM are installed and on your path.

#### To run the tests against the Go or Node filewatcher
```
git clone https://github.com/eclipse/codewind-filewatchers
cd codewind-filewatchers/Tests

# Run the tests again the Go filewatcher (this step will build the go filewatcher, build the tests, and then run them both)
./run_tests_go_filewatcher.sh

# Run the tests against the Node filewatcher (this step will build the node filewatcher, build the tests, and then run them both)
./run_tests_node_filewatcher.sh
```

#### To run the tests against the Java filewatcher:
The Java filewatcher tests are slightly different from above, only because the Java filewatcher is hosted in a different GitHub repository (*codewind-eclipse*) from the other 2 filewatchers.

```
git clone https://github.com/eclipse/codewind-eclipse
cd codewind-eclipse/dev/org.eclipse.codewind.filewatchers.core
mvn install
cd ../org.eclipse.codewind.filewatchers.standalonenio/
mvn package
cd ../../..



git clone https://github.com/eclipse/codewind-filewatchers
cd codewind-filewatchers/Tests
./run_tests_java_filewatcher_on_target.sh "(path to codewind-eclipse repo from above)"


```


## Developing the tests

Projects in this directory may be imported into Eclipse as Java projects. (Since every project listed below is just a Maven project, it is possible to import these into any IDE, but that is out of scope for these instructions). More information on building the filewatchers [is available here](https://github.com/eclipse/codewind-filewatchers/blob/master/DEVELOPING.md).

#### A) Import the following projects into your Eclipse workspace:
- FilewatcherTests
- MockCwctlSync
- org.eclipse.codewind.filewatchers.core (from https://github.com/eclipse/codewind-eclipse)
- org.eclipse.codewind.filewatchers.standalonenio  (from https://github.com/eclipse/codewind-eclipse)


#### B) You will need to start the filewatcher you want to test, before you start the tests. 

To run the Java filewatcher from Eclipse:

Create a Run entry to run the Java filwatcher. Select Run (menu item) > Run Configurations. Create a new Java Application Entry.

- In 'Main' tab, **Project**: `org.eclipse.codewind.filewatchers.standalonenio`, **Main class**: `org.eclipse.codewind.filewatchers.Main`
- In 'Arguments' tab, under **Program arguments**: `http://localhost:9090`
- In 'Environment' tab:
  - **Variable**: `CODEWIND_URL_ROOT`, **Value**: `http://localhost:9090`
  - **Variable**: `MOCK_CWCTL_INSTALLER_PATH`, **Value**: `(path to git repo)/codewind-filewatchers/MockCwctlSync/target/MockCwctlSync-0.0.1-SNAPSHOT.jar`
- Click Apply.
- Click Run.


#### C) To run the tests from Eclipse:
1. First we build the MockCwctlSync utility described above. Do the following from the command line:
```
cd (path to git repo)/codewind-filewatchers/MockCwctlSync
mvn package
```
2. Back in Eclipse, right click on `FilewatcherTests.java` and select `Run as > JUnit Test`. The filewatcher test will wait for the filewatcher from `B)` to connect to it, and once connected the tests will start. The tests take about 15 minutes to run, and you should see results in the JUnit view.
3. Once the tests complete, make sure you have killed both the test process and the filewatcher process.



## Debugging the tests

To debug the tests from Eclipse, just select the `Debug As` menu item, rather than `Run As`, in Eclipse, when executing the tests.


## FAQ


### I see one or more of the following while running the tests; is it expected?


#### This Jetty exception is expected; this is the message Jetty prints when the WebSocket is closed (I have not found a way to supress it). It can be safely ignored:
```
2020-04-15 19:51:41.001:WARN:oejwcec.CompressExtension:Thread-491:
java.lang.NullPointerException: Deflater has been closed
        at java.util.zip.Deflater.ensureOpen(Deflater.java:559)
        at java.util.zip.Deflater.deflate(Deflater.java:440)
        at org.eclipse.jetty.websocket.common.extensions.compress.CompressExtension$Flusher.compress(CompressExtension.java:488)
```

#### Lines that begin with `! ERROR !` are nearly always expected:

These are expected, especially lines that indicate that the filewatcher couldn't connect to the server, such as `WebSocket connection closed`, `Unable to establish connection to web socket on attempt`, `GET request failed.`, etc. 

If you look at the error contents you will see details of the error, but generally these errors are either from specific error conditions that the tests themselves are reporting, OR are related to the time between tests when the Jetty-based framework server has not yet restarted.

#### Lines that begin with `!!! SEVERE !!!` are NOT expected, with the only exception to this rule being this one `!!! SEVERE !!!  Asked to invoke CLI on a project that wasn't in the projects map`:

This 'asked to invoke CLI' exception is often printed while the tests are running: This is just due to how the test are written, and is expected during a test run. It may be safely ignored.

This message is truly SEVERE when the product is actually running in production, but is just an ERROR when running during automated tests, so I have intentionally left the severity at SEVERE for the former scenario.
