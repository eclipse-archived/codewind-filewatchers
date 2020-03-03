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

import static org.junit.Assert.assertFalse;
import static org.junit.Assert.assertNotNull;
import static org.junit.Assert.assertNull;
import static org.junit.Assert.assertTrue;
import static org.junit.Assert.fail;

import java.io.File;
import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.Collections;
import java.util.HashMap;
import java.util.Iterator;
import java.util.List;
import java.util.UUID;
import java.util.stream.Collectors;

import org.eclipse.codewind.filewatchers.test.infrastructure.ChangedFileEntry;
import org.eclipse.codewind.filewatchers.test.infrastructure.ChangedFileEntry.EventType;
import org.eclipse.codewind.filewatchers.test.infrastructure.CodewindTestState;
import org.eclipse.codewind.filewatchers.test.infrastructure.CodewindTestUtils;
import org.eclipse.codewind.filewatchers.test.infrastructure.ServerControl;
import org.eclipse.codewind.filewatchers.test.json.ProjectToWatchJson;
import org.eclipse.codewind.filewatchers.test.json.ProjectToWatchJson.RefPathEntry;
import org.junit.Test;

public class FilewatcherTests extends AbstractTest {

	@Test
	public void testCreateAndDeleteASingleFile() {

		initializeServer();
		sendTestName();

		ProjectToWatchJson p1 = newProject();
		watcherState.addOrUpdateProject(p1);

		waitForWatcherSuccess(p1);

		List<String> directoriesCreated = new ArrayList<String>();

		String suffix = "moo/cow-" + UUID.randomUUID();
		createDir(suffix, p1);
		directoriesCreated.add(suffix);
		waitForTrue(() -> {
			return changeList.getAllChangedFileEntriesByProjectId(p1).stream()
					.anyMatch(e -> e.getEventType() == EventType.CREATE && e.getPath().contains(suffix));
		});

		assertFilesExist(p1, changeList.getAllChangedFileEntriesByProjectId(p1));

		String match = null;

		for (ChangedFileEntry cfe : changeList.getAllChangedFileEntriesByProjectId(p1)) {
			match = directoriesCreated.stream().filter(e -> cfe.getPath().endsWith(e)).findAny().orElse(null);

			if (match != null) {
				break;
			}

		}

		assertNotNull(match);

		changeList.clear();

		for (String dir : directoriesCreated) {
			deleteDir(dir, p1);
		}

		waitNoThrowable(() -> {

			List<ChangedFileEntry> changedFiles = changeList.getAllChangedFileEntriesByProjectId(p1);

			for (String dir : directoriesCreated) {

				if (!changedFiles.stream()
						.anyMatch(e -> e.getPath().endsWith(dir) && e.getEventType() == EventType.DELETE)) {
					fail("Unable to find a file change for " + dir);
				}
			}

		});

	}

	@Test
	public void testCreateModifyAndDeleteLotsOfFiles() throws IOException {

		initializeServer();
		sendTestName();

		ProjectToWatchJson p1 = newProject();

		watcherState.addOrUpdateProject(p1);

		waitForWatcherSuccess(p1);

		List<File> dirs = new ArrayList<>();
		List<File> files = new ArrayList<>();

		log.out("Creating in: " + p1.getLocalPathToMonitor());

		buildRandomDirectoryStructure(100, .20f, 0, p1.getLocalPathToMonitor(), dirs, files);
		createRandomDirectoryStructure(dirs, files);

		___status___("Make sure we can find create events for all the files that we created.");
		waitForEventsFromFileList(files, EventType.CREATE, p1);

		assertFilesExist(p1, changeList.getAllChangedFileEntriesByProjectId(p1));

		___status___("Randomly write to a file, and make sure it shows up in the change list.");
		for (int x = 0; x < 10; x++) {

			changeList.clear();

			File fileToWriteTo = random(files);

			createOrModifyFile(fileToWriteTo);

			waitNoThrowable(() -> {

				List<ChangedFileEntry> chchchchChanges = changeList.getAllChangedFileEntriesByProjectId(p1);
				boolean matchFound = chchchchChanges.stream().filter(e -> e.getEventType() == EventType.MODIFY)
						.map(g -> convertToAbsolute(p1, g)).anyMatch(e -> {
							return e.getPath().equals(fileToWriteTo.getPath());
						});
				assertTrue("Match not found: " + fileToWriteTo.getPath(), matchFound);

				assertFilesExist(p1, chchchchChanges);
			});

		}

		___status___("Delete the files we created.");
		for (File f : files) {

			if (!f.delete()) {
				fail("Unable to delete: " + f.getPath());
			}

		}

		___status___("Make sure we can find delete events for all the files that we deleted.");
		waitForEventsFromFileList(files, EventType.DELETE, p1);

		waitNoThrowable(() -> {
			assertFilesExist(p1, changeList.getAllChangedFileEntriesByProjectId(p1));
		});

	}

	@Test
	public void testCreateSpecificFilesInRandomPlaces() {

		initializeServer();
		sendTestName();

		ProjectToWatchJson p1 = newProject();

		List<String> extensions = Arrays.asList(new String[] { ".txt", ".xml", ".html" });

		// None of these should trigger on any actual files or directories
		p1.getIgnoredFilenames().addAll(Arrays.asList("*.not", "*.present"));
		p1.getIgnoredPaths().addAll(Arrays.asList("*/not/*", "*/present/*"));

		watcherState.addOrUpdateProject(p1);

		waitForWatcherSuccess(p1);

		List<File> dirs = new ArrayList<>();
		List<File> files = new ArrayList<>();

		log.out("Creating in: " + p1.getLocalPathToMonitor());

		buildRandomDirectoryStructure(20, 1f, 0, p1.getLocalPathToMonitor(), dirs, files);
		createRandomDirectoryStructure(dirs, files);
		assertTrue("There should not be any files created.", files.size() == 0);

		// Wait for all the new directories to be created (but ignore the root dir,
		// which is not created)
		waitForEventsFromFileList(
				dirs.stream().filter(e -> !e.equals(p1.getLocalPathToMonitor())).collect(Collectors.toList()),
				EventType.CREATE, p1);

		assertFilesExist(p1, changeList.getAllChangedFileEntriesByProjectId(p1));

		for (int x = 0; x < 50; x++) {

			String extension = random(extensions);

			File dir = random(dirs);

			File f = createOrModifyFile(new File(dir, "my-file" + extension));

			log.out("Created file: " + f.getPath());

			waitNoThrowable(() -> {

				List<ChangedFileEntry> chchchchChanges = changeList.getAllChangedFileEntriesByProjectId(p1);

				assertTrue("Unable to find " + f.getPath() + " in change list.",
						chchchchChanges.stream().filter(e -> e.getEventType() == EventType.CREATE)
								.map(e -> convertToAbsolute(p1, e)).anyMatch(e -> e.getPath().equals(f.getPath())));

				assertFilesExist(p1, chchchchChanges);

			});
		}

	}

	@Test
	public void testFilenameIgnore() {

		initializeServer();
		sendTestName();

		ProjectToWatchJson p1 = newProject();

		List<String> extensions = Arrays.asList(new String[] { ".txt", ".xml", ".html" });

		extensions.stream().map(e -> "*" + e).forEach(e -> {
			p1.getIgnoredFilenames().add(e);
		});

		watcherState.addOrUpdateProject(p1);

		waitForWatcherSuccess(p1);

		List<File> dirs = new ArrayList<>();
		List<File> files = new ArrayList<>();

		log.out("Creating in: " + p1.getLocalPathToMonitor());

		buildRandomDirectoryStructure(200, 1f, 0, p1.getLocalPathToMonitor(), dirs, files);
		createRandomDirectoryStructure(dirs, files);

		assertTrue("There should not be any files created.", files.size() == 0);

		// Wait for all the new directories to be created (but ignore the root dir
		// which is not created.)
		waitForEventsFromFileList(
				dirs.stream().filter(e -> !e.equals(p1.getLocalPathToMonitor())).collect(Collectors.toList()),
				EventType.CREATE, p1);

		assertFilesExist(p1, changeList.getAllChangedFileEntriesByProjectId(p1));

		for (int x = 0; x < 10; x++) {

			String extension = random(extensions);

			File dir = random(dirs);

			File f = createOrModifyFile(new File(dir, "my-file" + extension));

			log.out(x + ") Wait to ensure " + f.getName() + " event does not occur.");

			repeatForXMsecs(5 * 1000, () -> {
				boolean matchFound = changeList.getAllChangedFileEntriesByProjectId(p1).stream()
						.filter(e -> e.getEventType() == EventType.CREATE).map(e -> convertToAbsolute(p1, e))
						.anyMatch(e -> e.getPath().equals(f.getPath()));

				assertFalse("A filtered file was found: " + f.getPath(), matchFound);

			});

		}

	}

	@Test
	public void testCanDeleteAfterUnWatch() {
		initializeServer();
		sendTestName();

		ProjectToWatchJson p1 = newProject();

		watcherState.addOrUpdateProject(p1);

		waitForWatcherSuccess(p1);

		List<File> dirs = new ArrayList<>();
		List<File> files = new ArrayList<>();

		log.out("Creating in: " + p1.getLocalPathToMonitor());

		buildRandomDirectoryStructure(200, .33f, 0, p1.getLocalPathToMonitor(), dirs, files);
		createRandomDirectoryStructure(dirs, files);

		// Wait for all the new directories to be created (but ignore the root dir which
		// is not created)
		waitForEventsFromFileList(excludeParentDirFromList(dirs, p1), EventType.CREATE, p1);

		waitForEventsFromFileList(files, EventType.CREATE, p1);

		for (File f : files) {
			f.delete();
			optionalArtificialDelay();
		}
		waitForEventsFromFileList(files, EventType.DELETE, p1);

		List<File> dirsToDelete = new ArrayList<>(excludeParentDirFromList(dirs, p1));
		while (dirsToDelete.size() > 0) {

			for (Iterator<File> it = dirsToDelete.iterator(); it.hasNext();) {
				File f = it.next();
				f.delete();
				if (!f.exists()) {
					it.remove();
					optionalArtificialDelay();
				}
			}

		}

		waitForEventsFromFileList(excludeParentDirFromList(dirs, p1), EventType.DELETE, p1);

		boolean canDelete = p1.getLocalPathToMonitor().delete();
		assertTrue(canDelete);

		assertFalse(p1.getLocalPathToMonitor().exists());

	}

	@Test
	public void testCreateDeleteCreate() {
		initializeServer();
		sendTestName();

		ProjectToWatchJson p1 = newProject();

		watcherState.addOrUpdateProject(p1);

		waitForWatcherSuccess(p1);

		log.out("Creating in: " + p1.getLocalPathToMonitor());

		___status___("Create a file and a directory and wait for it");
		File fileToCreate = new File(p1.getLocalPathToMonitor(), "cows-on-parade.txt");
		createOrModifyFile(fileToCreate);

		waitForEventsFromFileList(Arrays.asList(fileToCreate), EventType.CREATE, p1);

		___status___("Delete and wait for confirm");
		deleteFile(fileToCreate.getName(), p1);
		waitForEventsFromFileList(Arrays.asList(fileToCreate), EventType.DELETE, p1);

		assertTrue("Unable to delete root directory", p1.getLocalPathToMonitor().delete());

		assertTrue("Root directory still exists after deletion", !p1.getLocalPathToMonitor().exists());

		CodewindTestUtils.sleep(10 * 1000 * CHAOS_ENGINEERING_MULTIPLIER);

		changeList.clear();

		___status___("Create the old dir, and see if any events are thrown");
		assertTrue("Unable to create directory, again", p1.getLocalPathToMonitor().mkdir());

		repeatForXMsecs(10 * 1000, () -> {
			List<ChangedFileEntry> l = changeList.getAllChangedFileEntriesByProjectId(p1);
			assertTrue("Unexpected entries in list: " + l, l.size() == 0);
		});

		___status___("Create a file under the old dir, and see if any events are thrown");
		createOrModifyFile(fileToCreate);
		repeatForXMsecs(10 * 1000, () -> {
			List<ChangedFileEntry> l = changeList.getAllChangedFileEntriesByProjectId(p1);
			assertTrue("Unexpected entries in list: " + l, l.size() == 0);
		});

		___status___("Cleanup");
		fileToCreate.delete();
		p1.getLocalPathToMonitor().delete();
		assertTrue("Root directory still exists after deletion", !p1.getLocalPathToMonitor().exists());
		repeatForXMsecs(10 * 1000, () -> {
			List<ChangedFileEntry> l = changeList.getAllChangedFileEntriesByProjectId(p1);
			assertTrue("Unexpected entries in list: " + l, l.size() == 0);
		});

		watcherState.clearAllProjects();

		___status___("Create a new project with same path as the old one, but a different id");
		p1.getLocalPathToMonitor().mkdirs();

		ProjectToWatchJson p2 = newProject(p1.getLocalPathToMonitor().getPath());
		watcherState.addOrUpdateProject(p2);
		waitForWatcherSuccess(p2);

		___status___("Create the same old file under the new project, and wait for thrown events");
		createOrModifyFile(fileToCreate);
		waitForEventsFromFileList(Arrays.asList(fileToCreate), EventType.CREATE, p2);

		changeList.clear();
		assertTrue(fileToCreate.delete());
		waitForEventsFromFileList(Arrays.asList(fileToCreate), EventType.DELETE, p2);

		assertTrue(p2.getLocalPathToMonitor().delete());

	}

	@Test
	public void testTwoProjectsSharingAPathDuringANetworkDisconnect() {
		// 1) Create project A
		// 2) WebSocket disconnects
		// 3) Create project B, delete project A (both using the same directory)
		// 4) Websocket reconnects
		// 5) Make a change in the project directory, change should be associated with b

		ServerControl serverControl = initializeServer();

		sendTestName();

		ProjectToWatchJson p1 = newProject();

		watcherState.addOrUpdateProject(p1);

		waitForWatcherSuccess(p1);

		log.out("Creating in: " + p1.getLocalPathToMonitor());

		___status___("Create a file and a directory and wait for it");
		File fileToCreate = new File(p1.getLocalPathToMonitor(), "cows-on-parade.txt");
		createOrModifyFile(fileToCreate);

		waitForEventsFromFileList(Arrays.asList(fileToCreate), EventType.CREATE, p1);

		___status___("Stopping the server and simulating a deletion and creation");
		serverControl.stopServer();

		assertTrue(fileToCreate.delete());
		assertTrue(p1.getLocalPathToMonitor().delete());

		watcherState.clearAllProjects();

		boolean mkdirs = p1.getLocalPathToMonitor().mkdirs();
		assertTrue(mkdirs);

		serverControl = new ServerControl();
		addDisposableResource(serverControl);
		serverControl.startServer();

		changeList.clear();

		___status___("Register the new project");
		ProjectToWatchJson p2 = newProject(p1.getLocalPathToMonitor().getPath());
		watcherState.addOrUpdateProject(p2);

		waitForWatcherSuccess(p2);

		___status___("Create the file again and verify that we detect the change.");
		createOrModifyFile(fileToCreate);
		waitForEventsFromFileList(Arrays.asList(fileToCreate), EventType.CREATE, p2);

		assertTrue(fileToCreate.delete());

		waitForEventsFromFileList(Arrays.asList(fileToCreate), EventType.DELETE, p2);

	}

	@Test
	public void testFilterChange() {
//		Filter update:
//		- Watch one thing
//		- Change the filter
//		- Watch a second thing
//		- Verify we get results from it

		initializeServer();

		sendTestName();

		___status___("Create directory structure");

		ProjectToWatchJson p1 = newProject();
		File dir1 = createDir("dir1", p1);
		File dir2 = createDir("dir2", p1);

		File fileInDir1 = new File(dir1, "file1");
		File fileInDir2 = new File(dir2, "file2");

		createOrModifyFile(fileInDir1);
		createOrModifyFile(fileInDir2);

		watcherState.addOrUpdateProject(p1);

		waitForWatcherSuccess(p1);

		___status___("Test unfiltered events");
		{
			repeatForXMsecs(3000, () -> {
				assertTrue(changeList.getAllChangedFileEntriesByProjectId(p1).size() == 0);
			});

			createOrModifyFile(fileInDir1);
			createOrModifyFile(fileInDir2);

			waitForEventsFromFileList(Arrays.asList(fileInDir1, fileInDir2), EventType.MODIFY, p1);
			changeList.clear();
		}

		for (boolean ignoredPaths : Arrays.asList(true, false)) {
			for (File dirToFilter : Arrays.asList(dir1, dir2)) {

				___status___("Test events when " + dirToFilter.getName() + " is filtered out by "
						+ (ignoredPaths ? "ignoredPaths" : "ignoredFilenames"));
				p1.getIgnoredPaths().clear();
				p1.getIgnoredFilenames().clear();

				if (ignoredPaths) {
					p1.getIgnoredPaths().add("/" + dirToFilter.getName() + "/*");
				} else {
					p1.getIgnoredFilenames().add(dirToFilter.getName());
				}
				p1.regenerateWatchId();

				watcherState.addOrUpdateProject(p1);
				waitForWatcherSuccess(p1);

				repeatForXMsecs(5000, () -> {
					assertTrue(changeList.getAllChangedFileEntriesByProjectId(p1).size() == 0);
				});

				File fileToIgnore = dirToFilter == dir1 ? fileInDir1 : fileInDir2;
				File fileToSee = dirToFilter == dir1 ? fileInDir2 : fileInDir1;

				createOrModifyFile(fileInDir1);
				createOrModifyFile(fileInDir2);
				waitForEventsFromFileList(Arrays.asList(fileToSee), EventType.MODIFY, p1);

				repeatForXMsecs(5000, () -> {

					List<ChangedFileEntry> chchchchchanges = changeList.getAllChangedFileEntriesByProjectId(p1);
					log.out("ch-ch-ch-ch-changes: " + chchchchchanges);
					assertFalse(chchchchchanges.stream().map(g -> convertToAbsolute(p1, g))
							.anyMatch(e -> e.getPath().equals(fileToIgnore.getPath())));

				});

				changeList.clear();
			}

		}

	}

	@Test
	public void testRandomFileWritesAcrossMultipleSimultaneousProjects() {

		initializeServer();

		sendTestName();

		int numProjectsToCreate = 10;

		___status___("Creating directory structure and projects");

		HashMap<String /* project id */, List<File>> projectToFiles = new HashMap<>();

		List<ProjectToWatchJson> projects = new ArrayList<>();
		for (int x = 0; x < numProjectsToCreate; x++) {
			ProjectToWatchJson ptw = newProject();
			projects.add(ptw);

			List<File> dirs = new ArrayList<>();
			List<File> files = new ArrayList<>();

			buildRandomDirectoryStructure(100, .20f, 0, ptw.getLocalPathToMonitor(), dirs, files);
			createRandomDirectoryStructure(dirs, files, 0l);
			projectToFiles.put(ptw.getProjectID(), files);

			watcherState.addOrUpdateProject(ptw);
			waitForWatcherSuccess(ptw);
		}

		___status___("Creating random files in random dirs");

		repeatForXMsecs(45 * 1000, 0, () -> {
			changeList.clear();

			ProjectToWatchJson p = random(projects);

			List<File> files = projectToFiles.get(p.getProjectID());

			if (files.size() == 0) {
				return;
			}

			List<File> filesModified = new ArrayList<>();

			while (filesModified.size() < 5) {
				File toModify = random(files);

				createOrModifyFile(toModify);

				filesModified.add(toModify);

			}

			waitForEventsFromFileList(filesModified, EventType.MODIFY, p);

		});

		___status___("Deleting all the files");

		while (true) {

			long count = projectToFiles.values().stream().flatMap(e -> e.stream()).count();
			if (count == 0) {
				break;
			}

			ProjectToWatchJson ptw = random(projects);

			List<File> fileListInProject = projectToFiles.get(ptw.getProjectID());
			if (fileListInProject == null) {
				continue;
			}
			changeList.clear();

			log.out("Files to delete: " + count);

			List<File> filesDeleted = new ArrayList<>();

			while (fileListInProject.size() > 0 && filesDeleted.size() < 20) {
				Collections.shuffle(fileListInProject);
				File fileToDelete = fileListInProject.remove(0);
				filesDeleted.add(fileToDelete);

				if (fileListInProject.size() == 0) {
					projectToFiles.remove(ptw.getProjectID());
				}

				assertTrue(fileToDelete.delete());
			}

			waitForEventsFromFileList(filesDeleted, EventType.DELETE, ptw);

		}

	}

	@Test
	public void testWatchDirectoryThatDoesntYetExist() {
		initializeServer();

		sendTestName();

		___status___("Create project, but delete project directory ");

		ProjectToWatchJson p1 = newProject();

		assertTrue(p1.getLocalPathToMonitor().delete());

		assertFalse(p1.getLocalPathToMonitor().exists());

		watcherState.addOrUpdateProject(p1);

		// Wait 10 seconds: if at any point the watcher informs us of success, then fail
		repeatForXMsecs(10 * 1000, () -> {

			assertNotNull(p1.getProjectID());
			assertNotNull(p1.getProjectWatchStateId());

			Boolean watchStatusReceived = CodewindTestState.getInstance().getWatcherState()
					.getWatchStatus(p1.getProjectID(), p1.getProjectWatchStateId());

			assertNull(watchStatusReceived);
		});

		___status___("Creating project directory");

		assertTrue(p1.getLocalPathToMonitor().mkdir());

		assertTrue(p1.getLocalPathToMonitor().exists());

		waitForWatcherSuccess(p1);

	}

	@Test
	public void testEnsurePathToFilterCorrectlyFiltersOutSubPaths() {
		initializeServer();

		sendTestName();

		___status___("Create project, but delete project directory ");

		ProjectToWatchJson p1 = newProject();

		p1.getIgnoredPaths().add("/target");

		File dir = createDir("target", p1);

		watcherState.addOrUpdateProject(p1);

		waitForWatcherSuccess(p1);

		changeList.clear();

		___status___("Create file under filtered directory");
		File newFile = new File(dir, "my-new-file");
		createOrModifyFile(newFile);

		repeatForXMsecs(5 * 1000, () -> {
			assertTrue(changeList.getAllChangedFileEntriesByProjectId(p1).size() == 0);
		});

		___status___("Delete file under filtered directory");
		newFile.delete();

		repeatForXMsecs(5 * 1000, () -> {
			assertTrue(changeList.getAllChangedFileEntriesByProjectId(p1).size() == 0);
		});

		___status___("Create new dir under filtered directory");
		File newDir = new File(dir, "dir2");
		assertTrue(newDir.mkdir());
		repeatForXMsecs(5 * 1000, () -> {
			assertTrue(changeList.getAllChangedFileEntriesByProjectId(p1).size() == 0);
		});

		___status___("Create new file under new dir under filtered directory");
		newFile = new File(newDir, "my-new-file");
		createOrModifyFile(newFile);
		repeatForXMsecs(5 * 1000, () -> {
			assertTrue(changeList.getAllChangedFileEntriesByProjectId(p1).size() == 0);
		});

		___status___("Delete new file under new dir under filtered directory");
		newFile.delete();
		repeatForXMsecs(5 * 1000, () -> {
			assertTrue(changeList.getAllChangedFileEntriesByProjectId(p1).size() == 0);
		});

		___status___("Delete new dir under filtered directory");
		newDir.delete();
		repeatForXMsecs(5 * 1000, () -> {
			assertTrue(changeList.getAllChangedFileEntriesByProjectId(p1).size() == 0);
		});

	}

	@Test
	public void testEveryLifecycleCombinationOfPathFilter() {
		// 1) Should we create a (project)/target dir before we start watching the
		// project? (Y/N)
		// - If so, create it.

		// 2) Should the (project)/target dir be filtered out before we start watching
		// the project? (Y/N)
		// - If so, add it to the filter

		// 3) Start watching the project

		// 4) Should the (project)/target dir exist before we change the filter? (Y/N)
		// - If so, create it.

		// 5) Should the (project)/target be filtered out before we change the filter?
		// (Y/N)
		// - If so, update the filter

		// 6) Update the filter as above.
		// 7) Create a new file under target/ and make sure we do/don't receive events
		// based on settings.

		// 8) Update the above file under target/ and make sure we do/don't receive
		// events based on settings.

		initializeServer();

		sendTestName();

		List<boolean[]> permutations = generateEveryPermutationOfFactors(4);

		// Uncomment this to test a specific combination when debugging.
//		if (true) {
//			// exists-prewatch !filtered-prewatch exists-pre-filter-change
//			// filtered-after-filter-changed
//			permutations = Arrays.asList(new boolean[] { true, false, true, true });
//		}

		int count = 0;

		for (boolean[] permutation : permutations) {
			count++;
			boolean F1_targetDirExistsBeforeProjectWatch = permutation[0];
			boolean F2_targetIsFilteredBeforeProjectWatch = permutation[1];
			boolean F3_targetDirExistsBeforeFilterChange = permutation[2];
			boolean F4_targetIsFilteredOutAfterFilterChange = permutation[3];

			String permType = "";
			{
				permType += F1_targetDirExistsBeforeProjectWatch ? "exists-prewatch" : "!exists-prewatch";
				permType += " ";
				permType += F2_targetIsFilteredBeforeProjectWatch ? "filtered-prewatch" : "!filtered-prewatch";
				permType += " ";
				permType += F3_targetDirExistsBeforeFilterChange ? "exists-pre-filter-change"
						: "!exists-pre-filter-change";
				permType += " ";
				permType += F4_targetIsFilteredOutAfterFilterChange ? "filtered-after-filter-changed"
						: "!filtered-after-filter-changed";
				permType += " ";
			}

			___status___(count + ") Testing permutation: " + permType);

			File targetDir = null;
			ProjectToWatchJson p1 = newProject();
			if (F1_targetDirExistsBeforeProjectWatch) {
				targetDir = createDir("target", p1);
			}

			if (F2_targetIsFilteredBeforeProjectWatch) {
				p1.getIgnoredPaths().add("/target");
			}

			___status___(count + ") Adding project to watch");

			watcherState.addOrUpdateProject(p1);
			waitForWatcherSuccess(p1);
			changeList.clear();

			___status___(count + ") Regenerating watch id, and updating project");

			if (!F3_targetDirExistsBeforeFilterChange && targetDir != null) {
				assertTrue("Unable to rmdir", targetDir.delete());

				if (!F2_targetIsFilteredBeforeProjectWatch) {
					waitForEventsFromFileList(Arrays.asList(targetDir), EventType.DELETE, p1);
					changeList.clear();
				}
			}

			p1.getIgnoredPaths().clear();
			if (F4_targetIsFilteredOutAfterFilterChange) {
				p1.getIgnoredPaths().add("/target");
			}
			p1.regenerateWatchId();
			watcherState.addOrUpdateProject(p1);
			waitForWatcherSuccess(p1);

			___status___(count + ") Create file and wait for appropriate event response.");

			changeList.clear();

			File toCreate;

			if (targetDir == null) {
				targetDir = createDir("target", p1);
			}

			if (!targetDir.exists()) {
				assertTrue(targetDir.mkdirs());
			}

			toCreate = new File(targetDir, "new-file.txt");

			createOrModifyFile(toCreate);

			if (F4_targetIsFilteredOutAfterFilterChange) {
				repeatForXMsecs(5 * 1000, () -> {
					assertTrue(changeList.getAllChangedFileEntriesByProjectId(p1).size() == 0);
				});

			} else {
				waitForEventsFromFileList(Arrays.asList(toCreate), EventType.CREATE, p1);
			}

			changeList.clear();

			___status___(count + ") Modify file and wait for appropriate event response.");
			createOrModifyFile(toCreate);

			if (F4_targetIsFilteredOutAfterFilterChange) {
				repeatForXMsecs(5 * 1000, () -> {
					assertTrue(changeList.getAllChangedFileEntriesByProjectId(p1).size() == 0);
				});

			} else {
				waitForEventsFromFileList(Arrays.asList(toCreate), EventType.MODIFY, p1);
			}

			changeList.clear();
		}

	}

	@Test
	public void testNewTest() throws IOException {
		initializeServer();

		sendTestName();

		File mainDir = Files.createTempDirectory("fw-test-").toFile();

		File p1File;
		ProjectToWatchJson p1;
		{
			File p1Dir = new File(mainDir, ".config");
			p1Dir.mkdirs();
			p1 = newProject(p1Dir.getPath());
			p1.setProjectID(UUID.randomUUID().toString());
			___status___("Create project #1 - " + p1.getProjectID());

			p1File = new File(p1.getLocalPathToMonitor(), "file1");
			createOrModifyFile(p1File);

			watcherState.addOrUpdateProject(p1);

			p1.setType("non-project");

			waitForWatcherSuccess(p1);

			changeList.clear();

			createOrModifyFile(p1File);

			waitForEventsFromFileList(Arrays.asList(p1File), EventType.MODIFY, p1);
		}

		{
			changeList.clear();
			File p2Dir = new File(mainDir, "goproj");
			p2Dir.mkdirs();
			ProjectToWatchJson p2 = newProject(p2Dir.getPath());
			p2.setProjectID(UUID.randomUUID().toString());

			___status___("Create project #2 - " + p2.getProjectID());

			watcherState.addOrUpdateProject(p2);
			waitForWatcherSuccess(p2);

			File p2File = new File(p2.getLocalPathToMonitor(), "file2");
			createOrModifyFile(p2File);
			waitForEventsFromFileList(Arrays.asList(p2File), EventType.CREATE, p2);

			createOrModifyFile(p1File);
			waitForEventsFromFileList(Arrays.asList(p1File), EventType.MODIFY, p1);

			___status___("Test complete.");
		}

	}

	@Test
	public void testWatcherRefPaths() throws IOException {
		initializeServer();

		sendTestName();

		// Test when the linked file src exists, at the onset of project watch
		Path externalTempDir = Files.createTempDirectory("fw-temp-dir");

		{
			Path newFile = Files.createFile(externalTempDir.resolve("my-file"));

			___status___("Creating project 1");

			ProjectToWatchJson p1 = newProject();

			p1.getRefPaths().add(new RefPathEntry(newFile.toString(), "my-file"));

			watcherState.addOrUpdateProject(p1);

			waitForWatcherSuccess(p1);

			changeList.clear();

			___status___("Modifying " + newFile);

			createOrModifyFile(newFile);

			waitForEventsFromFileList(Arrays.asList(newFile.toFile()), EventType.MODIFY, p1);

			___status___("Deleting " + newFile);
			Files.delete(newFile);

			waitForEventsFromFileList(Arrays.asList(newFile.toFile()), EventType.DELETE, p1);

		}

		// Test when the linked file src does not exist at the onset of project watch
		{

			Path newFile = externalTempDir.resolve("my-file2");

			___status___("Creating project 2");

			ProjectToWatchJson p2 = newProject();
			p2.getRefPaths().add(new RefPathEntry(newFile.toString(), "my-file"));

			watcherState.addOrUpdateProject(p2);

			waitForWatcherSuccess(p2);

			___status___("Creating " + newFile);

			createOrModifyFile(newFile);

			waitForEventsFromFileList(Arrays.asList(newFile.toFile()), EventType.CREATE, p2);

			___status___("Modifying " + newFile);

			createOrModifyFile(newFile);

			waitForEventsFromFileList(Arrays.asList(newFile.toFile()), EventType.MODIFY, p2);

			// then delete

			___status___("Deleting " + newFile);

			Files.delete(newFile);

			waitForEventsFromFileList(Arrays.asList(newFile.toFile()), EventType.DELETE, p2);

		}

		// Test when the linked file is under the a project path, and thus should be
		// ignored.
		{

			___status___("Creating project 3");

			ProjectToWatchJson p3 = newProject();

			Path newFile = Files.createFile(p3.getLocalPathToMonitor().toPath().resolve("new-file-3"));

			p3.getIgnoredFilenames().add("new-file-3");
			p3.getRefPaths().add(new RefPathEntry(newFile.toString(), "my-file"));

			watcherState.addOrUpdateProject(p3);
			changeList.clear();

			___status___("Modifying " + newFile);

			createOrModifyFile(newFile);

			repeatForXMsecs(10 * 1000, () -> {
				List<ChangedFileEntry> l = changeList.getAllChangedFileEntriesByProjectId(p3);
				assertTrue("Unexpected entries in list: " + l, l.size() == 0);
			});

			___status___("Deleting " + newFile);

			Files.delete(newFile);

			repeatForXMsecs(10 * 1000, () -> {
				List<ChangedFileEntry> l = changeList.getAllChangedFileEntriesByProjectId(p3);
				assertTrue("Unexpected entries in list: " + l, l.size() == 0);
			});

		}

	}

	/**
	 * Does the filewatcher correctly detect (and call cwctl) when the root of the
	 * project is deleted
	 */
	@Test
	public void testDeleteProjectRoot() throws IOException {
		initializeServer();
		sendTestName();

		ProjectToWatchJson p1 = newProject();
		watcherState.addOrUpdateProject(p1);

		waitForWatcherSuccess(p1);

		final Path toDelete = p1.getLocalPathToMonitor().toPath();

		___status___("Deleting directory: " + toDelete);

		Files.delete(toDelete);

		assertTrue(!Files.exists(toDelete));

		waitNoThrowable(() -> {

			// We should be able to detect the deletion event.
			List<ChangedFileEntry> changedFiles = changeList.getAllChangedFileEntriesByProjectId(p1);

			assertTrue(changedFiles.size() > 0);

			if (!changedFiles.stream()
					.anyMatch(e -> e.getPath().equalsIgnoreCase("/") && e.getEventType() == EventType.DELETE)) {
				fail("Unable to find a file change for " + toDelete);
			}

		});

	}

	private List<File> excludeParentDirFromList(List<File> dirs, ProjectToWatchJson p) {
		return dirs.stream().filter(e -> !e.equals(p.getLocalPathToMonitor())).collect(Collectors.toList());
	}

}
