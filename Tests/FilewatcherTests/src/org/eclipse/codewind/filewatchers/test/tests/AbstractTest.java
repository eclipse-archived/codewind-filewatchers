/*******************************************************************************
 * Copyright (c) 2020 IBM Corporation
 * All rights reserved. This program and the accompanying materials
 * are made available under the terms of the Eclipse Public License v2.0
 * which accompanies this distribution, and is available at
 * http://www.eclipse.org/legal/epl-v20.html
 *
 * Contributors:
 *     IBM Corporation - initial API and implementation
 *******************************************************************************/

package org.eclipse.codewind.filewatchers.test.tests;

import static org.junit.Assert.assertTrue;
import static org.junit.Assert.fail;

import java.io.File;
import java.io.FileWriter;
import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.Collections;
import java.util.HashMap;
import java.util.Iterator;
import java.util.List;
import java.util.Map;
import java.util.Map.Entry;
import java.util.Random;
import java.util.UUID;
import java.util.concurrent.TimeUnit;

import org.eclipse.codewind.filewatchers.test.infrastructure.ChangedFileEntry;
import org.eclipse.codewind.filewatchers.test.infrastructure.ChangedFileEntry.EventType;
import org.eclipse.codewind.filewatchers.test.infrastructure.ChaosEngineering;
import org.eclipse.codewind.filewatchers.test.infrastructure.CodewindTestLogger;
import org.eclipse.codewind.filewatchers.test.infrastructure.CodewindTestState;
import org.eclipse.codewind.filewatchers.test.infrastructure.CodewindTestUtils;
import org.eclipse.codewind.filewatchers.test.infrastructure.PostRequestListFromWatcher;
import org.eclipse.codewind.filewatchers.test.infrastructure.ServerControl;
import org.eclipse.codewind.filewatchers.test.infrastructure.WatcherState;
import org.eclipse.codewind.filewatchers.test.json.ProjectToWatchJson;
import org.junit.After;
import org.junit.Before;
import org.junit.Rule;
import org.junit.rules.TestName;

public class AbstractTest {

	protected static final CodewindTestLogger log = CodewindTestLogger.getInstance();

	private static long delayBetweenIOOperations = 50;

	/**
	 * When chaos engineering mode is enabled, we need to wait longer during test
	 * delays. See ChaosEngineering class for details.
	 */
	static final int CHAOS_ENGINEERING_MULTIPLIER = ChaosEngineering.ENABLED ? 3 : 1;

	@Rule
	public TestName testName = new TestName();

	protected String getTestName() {
		return this.getClass().getSimpleName() + "." + testName.getMethodName();
	}

	@Before
	public void before() {

		log.out();
		log.out(getTestName() + " ----------------------------------------------------------- ");

	}

	@After
	public void after() {

		sendDebugMessage("Test " + getTestName() + " completed.");

		List<File> filesToDelete = new ArrayList<>();

		synchronized (toDispose_synch) {
			for (Object o : toDispose_synch) {
				if (o instanceof File) {
					filesToDelete.add((File) o);

				} else if (o instanceof ServerControl) {
					ServerControl sc = (ServerControl) o;
					sc.stopServer();
				} else {
					log.err("Unrecognized resource to dispose: " + o);
				}
			}
		}

		int filesToDeleteTotal = filesToDelete.size();
		log.out("* Deleting " + filesToDeleteTotal + " files. ");

		int attempts = 0;
		while (filesToDelete.size() > 0 && attempts < 50) {
			for (Iterator<File> it = filesToDelete.iterator(); it.hasNext();) {
				File f = it.next();

				if (f.exists()) {
					f.delete();
				}

				if (!f.exists()) {
					it.remove();
				}
			}
			attempts++;
		}

		// Wait 10 seconds after a test run to allow cleanup operations in the
		// filewatcher to occur (this causes issues when running on Jenkins due to
		// presumed slower I/O vs a laptop SSD).
		CodewindTestUtils.sleep(10 * 1000);
	}

	final PostRequestListFromWatcher changeList = CodewindTestState.getInstance().getChangeListFromWatcher();
	final WatcherState watcherState = CodewindTestState.getInstance().getWatcherState();

	private final List<Object> toDispose_synch = new ArrayList<>();

	ServerControl initializeServer() {

		changeList.clear();
		watcherState.clearAllProjects();

		ServerControl sc = new ServerControl();

		sc.startServer();

		addDisposableResource(sc);

		sc.waitForConnectedWebSocket();
		return sc;
	}

	void addDisposableResource(Object o) {
		synchronized (toDispose_synch) {
			toDispose_synch.add(o);
		}
	}

	void waitNoThrowable(ITestExecutionBlock t) {
		// The default wait time depends on whether chaos engineering is enabled
		int seconds = 30 * CHAOS_ENGINEERING_MULTIPLIER;
		long expireTimeInNanos = System.nanoTime() + TimeUnit.NANOSECONDS.convert(seconds, TimeUnit.SECONDS);

		Throwable lastThrowable = null;

		while (System.nanoTime() < expireTimeInNanos) {

			try {
				t.execute();
				return;
			} catch (Throwable th) {
				lastThrowable = th;
				/* ignore */
			}

			CodewindTestUtils.sleep(1000);
		}

		CodewindTestUtils.throwAsUnchecked(lastThrowable);

	}

	void waitForTrue(ITestCondition t) {
		waitForTrue(null, t);

	}

	void waitForTrue(String failReason, ITestCondition t) {
		// The default wait time depends on whether chaos engineering is enabled
		int seconds = 30 * CHAOS_ENGINEERING_MULTIPLIER;
		long expireTimeInNanos = System.nanoTime() + TimeUnit.NANOSECONDS.convert(seconds, TimeUnit.SECONDS);

		while (System.nanoTime() < expireTimeInNanos) {

			try {
				if (t.isTrue()) {
					return;
				}
			} catch (Throwable th) {
				/* ignore */
			}

			CodewindTestUtils.sleep(1000);
		}

		fail("waitForTrue timed out: " + (failReason != null ? failReason : ""));

	}

	Path createOrModifyFile(Path p) {
		File result = createOrModifyFile(p.toFile());
		return result.toPath();

	}

	File createOrModifyFile(File f) {
		try {
			log.out("Modifying file: " + f.getPath());
			boolean exists = f.exists();
			FileWriter fw;
			if (!f.getParentFile().exists()) {
				fail("Parent does not exist: " + f.getParentFile().getPath());
			}
			fw = new FileWriter(f);
			fw.write(UUID.randomUUID().toString());
			fw.close();

			if (!exists) { // this is correct
				addDisposableResource(f);
			}

			return f;
		} catch (IOException e) {
			throw new Error(e);
		}
	}

	public static void setDelayBetweenIOOperations(long delayBetweenIOOperations) {
		AbstractTest.delayBetweenIOOperations = delayBetweenIOOperations;
	}

	/** Introduce an optional artificial delay between I/O operations. */
	void optionalArtificialDelay() {
		if (delayBetweenIOOperations > 0) {
			CodewindTestUtils.sleep(delayBetweenIOOperations);
		}
	}

	File createDir(String name, ProjectToWatchJson project) {
		File rootDir = new File(project.getLocalPathToMonitor(), name);

		addDisposableResource(rootDir);

		log.out("Creating directory: " + name);
		if (!rootDir.mkdirs()) {
			fail("Unable to create: " + name);
		}

		return rootDir;
	}

	File deleteDir(String name, ProjectToWatchJson project) {
		File rootDir = new File(project.getLocalPathToMonitor(), name);

		assertTrue("Directory does not exist to be deleted: " + rootDir.getPath(), rootDir.exists());

		log.out("Deleting directory: " + name);
		boolean result = rootDir.delete();
		assertTrue("Unable to delete dir: " + rootDir.getPath(), result);

		return rootDir;
	}

	File deleteFile(String name, ProjectToWatchJson project) {
		File fileToDelete = new File(project.getLocalPathToMonitor(), name);

		assertTrue("File does not exist to be deleted: " + fileToDelete.getPath(), fileToDelete.exists());

		log.out("Deleting file: " + name);
		boolean result = fileToDelete.delete();
		assertTrue("Unable to delete file: " + fileToDelete.getPath(), result);

		return fileToDelete;
	}

	void deleteFile(File f) {
		boolean deleteResult = f.delete();

		if (f.exists()) {
			fail("File still exists after deletion:" + f.getName() + ", delete result was " + deleteResult);
		}
	}

	void waitForWatcherSuccess(ProjectToWatchJson ptw) {

		log.out("Waiting for watcher success: " + ptw.getProjectID() + " (" + ptw.getProjectWatchStateId() + ")");

		waitForTrue("Project watch state was never true: " + ptw.getProjectID() + " " + ptw.getProjectWatchStateId(),
				() -> CodewindTestState.getInstance().getWatcherState().getWatchStatus(ptw.getProjectID(),
						ptw.getProjectWatchStateId()) == true);

		log.out("Watcher success received: " + ptw.getProjectID() + " (" + ptw.getProjectWatchStateId() + ")");

		log.out("Post-watch success sleep.");
		CodewindTestUtils.sleep(CHAOS_ENGINEERING_MULTIPLIER * 5 * 1000);

	}

	ChangedFileEntry convertToAbsolute(ProjectToWatchJson ptw, ChangedFileEntry e) {

		File absolutePath = ChangedFileEntry.convertChangeEntryPathToAbsolute(ptw.getLocalPathToMonitor(), e.getPath());

		ChangedFileEntry result = new ChangedFileEntry(e.getEventType(), e.getTimestamp(), absolutePath.getPath(),
				e.isDirectory());

		return result;
	}

	ProjectToWatchJson newProject(String path) {
		File tempDir = new File(path);
		if (!tempDir.exists()) {
			fail("Directory should already exist: " + path);
		}

		if (File.separator.equals("\\")) {
			path = path.replace("\\", "/");

			path = "/c/" + path.substring(3);

		}

		ProjectToWatchJson result = new ProjectToWatchJson();
		result.setLocalPathToMonitor(tempDir);
		result.setPathToMonitor(path);
		result.setProjectID(UUID.randomUUID().toString());
		result.setProjectWatchStateId(UUID.randomUUID().toString());
		result.setType(random(Arrays.asList("non-project", "project")));
//		result.setProjectCreationTime(2573787985962l);
		return result;

	}

	ProjectToWatchJson newProject() {

		String path;
		File tempDir;
		try {

			tempDir = Files.createTempDirectory("fw-test-").toFile();
			tempDir.mkdirs();

			path = tempDir.getPath();

			return newProject(path);
		} catch (IOException e) {
			throw new RuntimeException(e);
		}
	}

	void createRandomDirectoryStructure(List<File> directories, List<File> files) {
		this.createRandomDirectoryStructure(directories, files, null);
	}

	void createRandomDirectoryStructure(List<File> directories, List<File> files, Long overrideArtificialDelay) {

		Collections.shuffle(directories);

		// Sort ascending by the number of / in the path
		sortDirectoriesAscendingBySlashes(directories);

		for (File dir : directories) {
			if (!dir.exists() && !dir.mkdirs()) {
				throw new RuntimeException("Unable to create directory: " + dir.getPath());
			}
			log.out("Created: " + dir);
			addDisposableResource(dir);

			if (overrideArtificialDelay != null) {
				CodewindTestUtils.sleep(overrideArtificialDelay);
			} else {
				optionalArtificialDelay();
			}

		}

		for (File file : files) {

			try {

				Files.createFile(file.toPath());
				log.out("Created: " + file);
				addDisposableResource(file);

				if (overrideArtificialDelay != null) {
					CodewindTestUtils.sleep(overrideArtificialDelay);
				} else {
					optionalArtificialDelay();
				}

			} catch (IOException e) {
				CodewindTestUtils.throwAsUnchecked(e);
				return;
			}

		}

	}

	/** Sort ascending by the number of / in the path */
	void sortDirectoriesAscendingBySlashes(List<File> dirList) {
		Collections.sort(dirList, (a, b) -> {
			int slashCountA = 0;
			int slashCountB = 0;

			for (int x = 0; x < a.getPath().length(); x++) {
				if (a.getPath().charAt(x) == File.separator.charAt(0)) {
					slashCountA++;
				}
			}

			for (int x = 0; x < b.getPath().length(); x++) {
				if (b.getPath().charAt(x) == File.separator.charAt(0)) {
					slashCountB++;
				}
			}

			return slashCountA - slashCountB;
		});

	}

	void buildRandomDirectoryStructure(int numFiles, float newDirPercent, long randomNumberSeed, File parent,
			List<File> outDirectories, List<File> outFiles) {

		Random r = new Random();

		if (randomNumberSeed != 0) {
			r = new Random(randomNumberSeed);
		}

		outDirectories.add(parent);

		for (int x = 0; x < numFiles; x++) {

			if (r.nextDouble() < newDirPercent) {
				// Create a new directory under an existing directory

				File parentDir = outDirectories.get(r.nextInt(outDirectories.size()));

				outDirectories.add(new File(parentDir, "d" + x));

			} else {
				// Create a new file under an existing directory

				File parentDir = outDirectories.get(r.nextInt(outDirectories.size()));
				outFiles.add(new File(parentDir, "f" + x));
			}

		}

	}

	/**
	 * Verify that files in the change list exist on the drive (unless they were
	 * deleted) and that the 'isDirectory' property for each change is accurate.
	 */
	void assertFilesExist(ProjectToWatchJson ptw, List<ChangedFileEntry> cfeParam) {

		Map<String, List<ChangedFileEntry>> entriesForAPath = new HashMap<>();

		// Sort descending by timestamp
		cfeParam.sort((a, b) -> {
			long val = (b.getTimestamp() - a.getTimestamp());

			if (val > 0) {
				return 1;
			}
			if (val < 0) {
				return -1;
			}

			return 0;

		});

		cfeParam.forEach(e -> {
			List<ChangedFileEntry> l = entriesForAPath.computeIfAbsent(e.getPath(), f -> new ArrayList<>());
			l.add(e);
		});

		for (Iterator<Entry<String, List<ChangedFileEntry>>> it = entriesForAPath.entrySet().iterator(); it
				.hasNext();) {

			Entry<String, List<ChangedFileEntry>> e = it.next();

			// Find the timestamp of the last entry in the list (the most recevent event).
			// Find any entries that have that timestamp (may be >1).
			// If one of those entries is a delete, then the file has been deleted and we
			// don't need to check it here.
			ChangedFileEntry lastEntry = e.getValue().get(0);
			outer: for (ChangedFileEntry cfe : e.getValue()) {

				if (cfe.getTimestamp() == lastEntry.getTimestamp()) {
					if (cfe.getEventType() == EventType.DELETE) {
						it.remove();
						break outer;
					}
				}

			}

		}

		for (Map.Entry<String, List<ChangedFileEntry>> e : entriesForAPath.entrySet()) {

			List<ChangedFileEntry> currList = e.getValue();

			for (ChangedFileEntry cfe : currList) {

				ChangedFileEntry convertedCfe = convertToAbsolute(ptw, cfe);

				File f = new File(convertedCfe.getPath());

				assertTrue("File does not exist: " + f.getPath(), f.exists());
				assertTrue("Directory status for file does not match: " + f.getPath() + " " + f.isDirectory() + " "
						+ cfe.isDirectory(), f.isDirectory() == cfe.isDirectory());

			}

		}

//		entriesForAPath.values().stream().flatMap(e -> e.stream()).map(e -> convertToAbsolute(ptw, e)).forEach(e -> {
//			File f = new File(e.getPath());
//			try {
//				assertTrue("File does not exist: " + f.getPath(), f.exists());
//				assertTrue("Directory status for file does not match: " + f.getPath(),
//						f.isDirectory() == e.isDirectory());
//			} catch (Throwable t) {
//
//				List<ChangedFileEntry> l = entriesForAPath.get(e.getPath());
//				System.out.println(l);
//
//			}
//		});

	}

	void waitForEventsFromFileList(List<File> filesWaitingFor, EventType eventTypeParam, ProjectToWatchJson p1) {
		waitNoThrowable(() -> {

			List<File> missing = new ArrayList<>();

			boolean pass = true;

			int matches = 0;
			for (File fileWaitingFor : filesWaitingFor) {

				List<ChangedFileEntry> allChangedFileEntries = changeList.getAllChangedFileEntriesByProjectId(p1);

				// Find a create event matching 'f'
				boolean matchFound = allChangedFileEntries.stream().filter(e -> e.getEventType() == eventTypeParam)
						.map(g -> convertToAbsolute(p1, g)).anyMatch(e -> {

							return Paths.get(e.getPath()).normalize().equals(fileWaitingFor.toPath().normalize());

						});

				if (matchFound) {
					matches++;
				} else {
					missing.add(fileWaitingFor);
					pass = false;
				}

			}

			if (!pass) {
				fail("Some " + eventTypeParam.name() + " matches not found: " + matches + "/" + filesWaitingFor.size()
						+ " Missing: " + missing);

			} else {
				System.out.println("Succcessfully matched " + matches + " files.");
			}
		});

	}

	/**
	 * This method intentionally has a weird name, to make it stand out in the test
	 * source code.
	 */
	public void ___status___(String status) {
		String testName = getTestName();
		CodewindTestState.getInstance().getConnectionState().ifPresent(e -> {
			e.sendDebugMessage(testName + " - Status: " + status);
		});

		log.out();
		log.out("[" + testName + "] * " + status);
		log.out();
	}

	static void sendDebugMessage(String msg) {
		CodewindTestState.getInstance().getConnectionState().ifPresent(e -> {
			e.sendDebugMessage(msg);
		});
	}

	void sendTestName() {
		String testName = getTestName();
		sendDebugMessage("Starting test " + testName);
	}

	/** Try once per second, and immediately throw any errors (no retries). */
	void repeatForXMsecs(long howLongToWaitInMsecs, Runnable r) {

		this.repeatForXMsecs(howLongToWaitInMsecs, 1000, r);
	}

	void repeatForXMsecs(long howLongToWaitInMsecs, long delayBetweenRunsInMsecs, Runnable r) {

		howLongToWaitInMsecs = howLongToWaitInMsecs * CHAOS_ENGINEERING_MULTIPLIER;

		long expireTimesInNanos = System.nanoTime()
				+ TimeUnit.NANOSECONDS.convert(howLongToWaitInMsecs, TimeUnit.MILLISECONDS);

		while (System.nanoTime() < expireTimesInNanos) {

			r.run();

			CodewindTestUtils.sleep(delayBetweenRunsInMsecs);

		}

	}

	List<boolean[]> generateEveryPermutationOfFactors(int numFactors) {

		List<boolean[]> result = new ArrayList<>();

		int maxVal = (int) Math.pow(2, numFactors);

		for (int x = 0; x < maxVal; x++) {

			String str = Integer.toBinaryString(x);

			while (str.length() < numFactors) {
				str = "0" + str;
			}
			boolean[] barr = new boolean[numFactors];

			for (int y = 0; y < str.length(); y++) {

				barr[y] = str.charAt(y) == '1';

			}
			result.add(barr);

		}

		return result;

	}

	public <T> T random(List<T> list) {
		return list.get((int) (list.size() * Math.random()));
	}

	interface ITestExecutionBlock {
		public void execute();
	}

	interface ITestCondition {
		public boolean isTrue();
	}
}
