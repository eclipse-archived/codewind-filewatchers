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

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

import org.eclipse.codewind.filewatchers.test.infrastructure.ChangedFileEntry.EventType;

public class PostRequestContent {
	private final String projectId;
	private final long timestamp;
	private final List<ChangedFileEntry> changedFileEntries;

	public PostRequestContent(String projectId, long timestamp, List<ChangedFileEntry> changedFileEntries) {
		this.projectId = projectId;
		this.timestamp = timestamp;
		this.changedFileEntries = changedFileEntries;
	}

	public String getProjectId() {
		return projectId;
	}

	public long getTimestamp() {
		return timestamp;
	}

	public List<ChangedFileEntry> getChangedFileEntries() {
		return changedFileEntries;
	}

	/**
	 * A duplicate is defined as two CREATE/DELETE entries in row, for a given path.
	 */
	public boolean containsDuplicates() {

		if (changedFileEntries == null || changedFileEntries.size() == 0) {
			return false;
		}

		Map<String /* path */, List<ChangedFileEntry>> pathToChanges = new HashMap<>();

		for (ChangedFileEntry e : changedFileEntries) {
			String path = e.getPath();
			List<ChangedFileEntry> cfeList = pathToChanges.computeIfAbsent(path, f -> new ArrayList<>());
			cfeList.add(e);
		}

		for (Map.Entry<String, List<ChangedFileEntry>> me : pathToChanges.entrySet()) {

			List<ChangedFileEntry> changes = me.getValue();

			for (int x = 0; x < changes.size() - 1; x++) {

				ChangedFileEntry curr = changes.get(x);
				ChangedFileEntry next = changes.get(x + 1);

				if (curr.getEventType() != EventType.DELETE && curr.getEventType() != EventType.CREATE) {
					continue;
				}

				if (curr.getEventType() == next.getEventType()) {
					return true;
				}
			}

		}

		return false;

	}
}
