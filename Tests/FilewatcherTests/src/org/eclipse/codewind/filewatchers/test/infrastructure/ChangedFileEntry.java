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

package org.eclipse.codewind.filewatchers.test.infrastructure;

import java.io.File;

import org.eclipse.codewind.filewatchers.test.json.ChangedFileEntryJson;

public class ChangedFileEntry {

	public static enum EventType {
		CREATE, MODIFY, DELETE
	}

	private final EventType eventType;

	private final long timestamp;

	private final String path;

	private final boolean directory;

	public ChangedFileEntry(ChangedFileEntryJson j) {
		this.timestamp = j.getTimestamp();
		this.path = j.getPath();
		this.directory = j.isDirectory();

		if (j.getType().equals("CREATE")) {
			this.eventType = EventType.CREATE;
		} else if (j.getType().equals("MODIFY")) {
			this.eventType = EventType.MODIFY;
		} else if (j.getType().equals("DELETE")) {
			this.eventType = EventType.DELETE;
		} else {
			throw new IllegalArgumentException();
		}
	}

	public ChangedFileEntry(EventType eventType, long timestamp, String path, boolean isDirectory) {
		this.eventType = eventType;
		this.timestamp = timestamp;
		this.path = path;
		this.directory = isDirectory;
	}

	public EventType getEventType() {
		return eventType;
	}

	public long getTimestamp() {
		return timestamp;
	}

	public String getPath() {
		return path;
	}

	public boolean isDirectory() {
		return directory;
	}

	public static File convertChangeEntryPathToAbsolute(File parent, String path) {

		// This is hack, but works fine here in the context of an automated test.
		if (path.startsWith("/home") || path.toLowerCase().startsWith("/users")
				|| path.toLowerCase().startsWith("c:\\users") || path.toLowerCase().startsWith("/tmp")) {
			return new File(path);
		}

		path = path.replace("/", File.separator);

		return new File(parent, path);

	}

	@Override
	public String toString() {

		return "[" + eventType.name() + "] " + path + " @ " + timestamp;
	}
}
