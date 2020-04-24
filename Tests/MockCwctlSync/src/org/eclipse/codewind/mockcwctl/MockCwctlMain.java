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

package org.eclipse.codewind.mockcwctl;

import java.io.ByteArrayOutputStream;
import java.io.IOException;
import java.net.URI;
import java.net.URISyntaxException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.util.ArrayList;
import java.util.Base64;
import java.util.Collections;
import java.util.HashSet;
import java.util.Iterator;
import java.util.List;
import java.util.Map;
import java.util.concurrent.TimeUnit;
import java.util.regex.Pattern;
import java.util.stream.Collectors;
import java.util.zip.Deflater;
import java.util.zip.DeflaterOutputStream;

import javax.json.bind.JsonbBuilder;
import javax.json.bind.JsonbException;

import org.eclipse.codewind.mockcwctl.HttpUtil.HttpResult;
import org.eclipse.codewind.mockcwctl.json.ChangedFileEntryJson;
import org.eclipse.codewind.mockcwctl.json.FileChangeMsgJson;
import org.eclipse.codewind.mockcwctl.json.MockProjectWatchJson;
import org.eclipse.codewind.mockcwctl.json.ProjectContentsJson;
import org.eclipse.codewind.mockcwctl.json.ProjectContentsJson.FileEntry;

public class MockCwctlMain {

	private static Path _processOpenFile;

	public static void main(String[] args) throws IOException, URISyntaxException {

		killProcessAfterXMinutes(3);

		String projectId = null;
		{
			for (int x = 0; x < args.length; x++) {

				if (args[x].equals("-i")) {
					projectId = args[x + 1];

				}
			}

		}
		if (projectId == null) {
			throw new RuntimeException("Could not find project id");
		}

		Path homeDir = Paths.get(System.getProperty("user.home"), ".mockcwctl", "automated-tests", projectId);

		Path failFile = homeDir.resolve(".fail");

		if (Files.exists(failFile)) {
			System.err.println("Fail file detected: " + failFile);
			return;
		}

		// We use the presence of this file to detect when multiple cwctl utilities are
		// running at a time (which should not happen, and is a bug.)
		Path processOpenFile = homeDir.resolve("process-is-open");

		// If we detect multiple running utilities, we flag it as an error and create a
		// permanent fail file to cause other subsequent tests to fail.
		if (Files.exists(processOpenFile)) {
			System.err.println("Process open file detected.");

			Files.createFile(failFile);

			return;
		}

		Path parent = processOpenFile.getParent();
		if (!Files.exists(parent)) {
			Files.createDirectories(parent);
		}

		Files.createFile(processOpenFile);

		try {
			_processOpenFile = processOpenFile;
			innerMain(args);
		} finally {
			Files.delete(processOpenFile);
			Files.delete(homeDir);
		}

	}

	private static void innerMain(String[] args) throws JsonbException, IOException, URISyntaxException {

		final boolean DEBUG = false;

		String urlRoot = "http://localhost:9090";

		for (Map.Entry<String, String> e : System.getenv().entrySet()) {

			if (e.getKey().equalsIgnoreCase("codewind_url_root")) {
				urlRoot = e.getValue();
			}
		}

		Path homeDir = Paths.get(System.getProperty("user.home"), ".mockcwctl");

		String projectPathParam = null;
		String projectId = null;
		long latestTimestampParam = 0;
		String projectJsonStrBase64 = null;

		// Extract the parameters
		for (int x = 0; x < args.length; x++) {

			if (args[x].equals("-p")) {
				projectPathParam = args[x + 1];

			} else if (args[x].equals("-i")) {
				projectId = args[x + 1];

			} else if (args[x].equals("-t")) {
				latestTimestampParam = Long.parseLong(args[x + 1]);

			} else if (args[x].equals("-projectJson")) {
				projectJsonStrBase64 = args[x + 1].trim();
			}

		}

		final long latestTimestamp = latestTimestampParam;

		if (projectPathParam == null || projectId == null || projectJsonStrBase64 == null) {
			System.err.println("Missing value.");
			return;
		}

		String projectJsonReceived = new String(Base64.getDecoder().decode(projectJsonStrBase64));

		System.out.println("projectJsonReceived: " + projectJsonReceived);

		MockProjectWatchJson pwJson = JsonbBuilder.create().fromJson(projectJsonReceived, MockProjectWatchJson.class);
		if (pwJson.getFilesToWatch() == null) {
			pwJson.setFilesToWatch(Collections.emptyList());
		}
		if (pwJson.getIgnoredFilenames() == null) {
			pwJson.setIgnoredFilenames(Collections.emptyList());
		}
		if (pwJson.getIgnoredPaths() == null) {
			pwJson.setIgnoredPaths(Collections.emptyList());
		}

		final String projectPath = projectPathParam;

		Path tempDir = homeDir.resolve(projectId);

		// This mock cwctl utility maintains a list of the previous state of the file
		// system when the utility was last run. This allows it to detect changed,
		// modified, and deleted files.
		//
		// This previous file system state is stored as JSON in this file:
		Path previousStateJsonPath = tempDir.resolve("previous-state.json");
		boolean previousStateExists = false;
		ProjectContentsJson previousState = new ProjectContentsJson();
		if (Files.exists(previousStateJsonPath)) {
			previousState = JsonbBuilder.create().fromJson(Files.newBufferedReader(previousStateJsonPath),
					ProjectContentsJson.class);
			previousStateExists = true;
		}

		Files.createDirectories(tempDir);

		List<FileEntry> deletedFilesOrFolders = previousState.getEntries().stream()
				.filter(e -> !Files.exists(Paths.get(e.getPath()))).collect(Collectors.toList());

		// If the root project directory doesn't exist, record that as a deletion.
		if (!Files.exists(Paths.get(projectPathParam))) {
			FileEntry fe = new FileEntry();
			fe.setDirectory(Files.isDirectory(Paths.get(projectPathParam)));
			fe.setModification(System.currentTimeMillis());
			fe.setPath(projectPath);
			deletedFilesOrFolders.add(fe);
		}

		List<WalkEntry> allFiles = walkDirectory(Paths.get(projectPathParam));
		{
			for (String watchedFile : pwJson.getFilesToWatch()) {
				Path watchedFilePath = Paths.get(watchedFile);
				if (Files.exists(watchedFilePath)) {
					allFiles.add(new WalkEntry(watchedFilePath, Files.getLastModifiedTime(watchedFilePath).toMillis()));
				}
			}

			if (DEBUG) {
				allFiles.forEach(e -> System.out.println("af:" + e.path));
			}

		}

		// If this is the first time project sync has run for this project, then just
		// write the filesystem state and return.
		if (!previousStateExists) {
			System.out.println("* Previous state doesn't exist, so writing database and returning.");
			if (allFiles.size() < 10) {
				allFiles.forEach(e -> {
					System.out.println("- " + e.path.toString());
				});
			}
			writeJsonDB(allFiles, previousStateJsonPath);
			return;
		}

		List<WalkEntry> modifiedFiles = new ArrayList<>(allFiles.parallelStream()
				.filter(e -> e.modificationTime > latestTimestamp).collect(Collectors.toList()));
		{
			if (DEBUG) {
				modifiedFiles.forEach(e -> System.out.println("mf:" + e.path));
			}
		}

		// Find files that were added (rather than modified) by comparing with the
		// database.
		List<WalkEntry> addedFiles = new ArrayList<>();
		// Files are only considered to be added once we have first observed the full
		// database structure (eg a previous state exists).

		HashSet<String> existingPathsFromDatabase = new HashSet<String>(
				previousState.getEntries().parallelStream().map(e -> e.getPath()).collect(Collectors.toList()));

		existingPathsFromDatabase.stream().forEach(e -> {
			if (DEBUG) {
				System.out.println("epfd: " + e);
			}
		});

		for (Iterator<WalkEntry> it = modifiedFiles.iterator(); it.hasNext();) {
			WalkEntry we = it.next();
			if (DEBUG) {
				System.out.println("we:" + we.path.toString() + "");
			}
			if (!existingPathsFromDatabase.contains(we.path.toString())) {
				it.remove();
				addedFiles.add(we);
			}
		}

		long requestTimestamp = System.currentTimeMillis();

		List<ChangedFileEntryJson> cfejList = new ArrayList<>();

		// Convert added/changed/deleted to a single JSON object
		{

			System.out.println("added:");
			addedFiles.forEach(e -> {
				System.out.println(e.path);
				ChangedFileEntryJson cfej = new ChangedFileEntryJson();
				cfej.setDirectory(Files.isDirectory(e.path));
				cfej.setPath(e.path.toString());
				cfej.setTimestamp(requestTimestamp);
				cfej.setType("CREATE");
				cfejList.add(cfej);

			});

			System.out.println("modified:");
			modifiedFiles.forEach(e -> {
				System.out.println(e.path);
				ChangedFileEntryJson cfej = new ChangedFileEntryJson();
				cfej.setDirectory(Files.isDirectory(e.path));
				cfej.setPath(e.path.toString());
				cfej.setTimestamp(requestTimestamp);
				cfej.setType("MODIFY");
				cfejList.add(cfej);
			});

			System.out.println("deleted:");
			deletedFilesOrFolders.forEach(e -> {
				System.out.println(e.getPath());
				ChangedFileEntryJson cfej = new ChangedFileEntryJson();
				cfej.setDirectory(e.isDirectory());
				cfej.setPath(e.getPath());
				cfej.setTimestamp(requestTimestamp);
				cfej.setType("DELETE");
				cfejList.add(cfej);
			});

			// Convert to Unix-style project relative path
			cfejList.forEach(e -> {
				String path = e.getPath();

				if (path.contains(projectPath)) {
					path = path.replace(projectPath, "");

					while (path.startsWith("/") || path.startsWith("\\")) {
						path = path.substring(1);
					}
					path = "/" + path.replace("\\", "/");
				}

				e.setPath(path);
			});
		}

		// Write JSON database for latest use
		writeJsonDB(allFiles, previousStateJsonPath);

		// Filter out using the provided filter.
		outer: for (Iterator<ChangedFileEntryJson> it = cfejList.iterator(); it.hasNext();) {
			ChangedFileEntryJson cfej = it.next();

			String path = cfej.getPath();

			System.out.println();
			System.out.println("path being processed: " + path);

			if (pwJson.getIgnoredFilenames().size() > 0) {

				Path pathProper = Paths.get(path);

				for (int x = 0; x < pathProper.getNameCount(); x++) {
					String filename = pathProper.getName(x).toString();

					for (String ignoredFilenameFilter : pwJson.getIgnoredFilenames()) {
						String filterText = ignoredFilenameFilter.replace("*", ".*");
						Pattern p = Pattern.compile(filterText);

						if (p.matcher(filename).matches()) {
							it.remove();
							continue outer;
						}
					}

				}

			}

			if (pwJson.getIgnoredPaths().size() > 0) {

				for (String ignoredPathFilter : pwJson.getIgnoredPaths()) {
					String filterText = ignoredPathFilter.replace("*", ".*");
					System.out.println("filter: " + filterText);
					Pattern p = Pattern.compile(filterText);

					if (p.matcher(path).matches()) {
						System.out.println("matched.");
						it.remove();
						continue outer;
					}

				}

			}

		}

		{
			postToURL(urlRoot, cfejList, projectId, requestTimestamp);

		}

	}

	// Post the changes to the provided URL (keep trying on fail)
	private static void postToURL(String urlRoot, List<ChangedFileEntryJson> cfejList, String projectId,
			long requestTimestamp) {
		FileChangeMsgJson fcmj = new FileChangeMsgJson();

		String base64 = Base64.getEncoder().encodeToString(compressString(JsonbBuilder.create().toJson(cfejList)));

		fcmj.setMsg(base64);

		String msgData = JsonbBuilder.create().toJson(fcmj);

		System.out.println("ts: " + requestTimestamp);

		boolean isGoodResponse = false;

		while (!isGoodResponse) {

			try {
				HttpResult result = HttpUtil.post(new URI(
						urlRoot + "/api/v1/projects/" + projectId + "/file-changes?timestamp=" + requestTimestamp),
						msgData, (conn -> {
							conn.setConnectTimeout(10 * 1000);
							conn.setReadTimeout(10 * 1000);
							HttpUtil.allowAllCerts(conn);
						}));
				isGoodResponse = result.isGoodResponse;
			} catch (Exception ce) {
				System.err.println("Received " + ce.getClass().getSimpleName());
				isGoodResponse = false;
			}

			if (!isGoodResponse) {
				System.err.println("POST failed, retrying....");
			}
			try {
				Thread.sleep(100);
			} catch (InterruptedException e1) {
				e1.printStackTrace();
			}
		}

	}

	private static void writeJsonDB(List<WalkEntry> allFiles, Path previousStateJson)
			throws JsonbException, IOException {
		// Write JSON database for latest use
		ProjectContentsJson pcj = new ProjectContentsJson();
		pcj.getEntries().addAll(allFiles.stream().map(e -> {
			FileEntry fe = new FileEntry();
			fe.setDirectory(Files.isDirectory(e.path));
			try {
				fe.setModification(Files.getLastModifiedTime(e.path).toMillis());
			} catch (IOException e1) {
				System.err.println("Exception on get modification - " + e1.getClass().getSimpleName() + ": "
						+ e1.getMessage() + ". Not updating the modification time.");
				fe.setModification(e.modificationTime);
			}
			fe.setPath(e.path.toString());
			return fe;
		}).filter(e -> e != null).collect(Collectors.toList()));

		Files.write(previousStateJson, JsonbBuilder.create().toJson(pcj).getBytes());
	}

	private static List<WalkEntry> walkDirectory(Path directoryParam) throws IOException {

		List<WalkEntry> result = new ArrayList<>();

		List<Path> directoriesToProcess = new ArrayList<>();

		if (Files.exists(directoryParam)) {
			directoriesToProcess.add(directoryParam);
		} else {
			System.err.println("Directory does not exist: " + directoryParam);
		}

		while (directoriesToProcess.size() > 0) {

			Path curr = directoriesToProcess.remove(0);

			Files.list(curr).forEach(e -> {
				try {

					result.add(new WalkEntry(e, Files.getLastModifiedTime(e).toMillis()));
				} catch (IOException e1) {
					throw new RuntimeException(e1);
				}

				if (Files.isDirectory(e)) {
					directoriesToProcess.add(e);
				}

			});

		}

		return result;

	}

	private static void killProcessAfterXMinutes(int minutes) {
		Thread t = new Thread() {
			public void run() {
				try {
					TimeUnit.MINUTES.sleep(minutes);
				} catch (InterruptedException e) {
					e.printStackTrace();
				}
				if (_processOpenFile != null) {
					try {
						Files.delete(_processOpenFile);
					} catch (IOException e) {
						e.printStackTrace();
					}
				}

				System.exit(0);
			};
		};
		t.setDaemon(true);
		t.start();

	}

	@SuppressWarnings("unused")
	private final static byte[] compressString(String str) {

		ByteArrayOutputStream baos = new ByteArrayOutputStream();

		DeflaterOutputStream dos = new DeflaterOutputStream(baos, new Deflater(Deflater.BEST_SPEED));

		int uncompressedSize;
		try {
			byte[] strBytes = str.getBytes();
			dos.write(strBytes);
			dos.close();
			baos.close();
		} catch (IOException e) {
			throw new RuntimeException(e);
		}

		byte[] result = baos.toByteArray();

		return result;
	}

	private static class WalkEntry {
		Path path;
		long modificationTime;

		public WalkEntry(Path path, long modificationTime) {
			this.path = path;
			this.modificationTime = modificationTime;
		}

	}

}
