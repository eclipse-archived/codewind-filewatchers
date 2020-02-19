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

import static org.junit.Assert.fail;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.stream.Collectors;

import org.eclipse.codewind.filewatchers.test.json.ProjectToWatchJson;

public class PostRequestListFromWatcher {

	private final List<PostRequestContent> postRequests_synch_lock = new ArrayList<>();

	private final Object lock = new Object();

	private HashMap<String /* project id */, Long> largestTimestampReceived_synch_lock = new HashMap<>();

	private boolean severeErrorOccurred_synch_lock = false;

	private boolean duplicatesDetected_synch_lock = false;

	private static final CodewindTestLogger log = CodewindTestLogger.getInstance();

	public PostRequestListFromWatcher() {
	}

	public void addChangedFileEntries(PostRequestContent postRequestContent) {
		synchronized (lock) {
			postRequests_synch_lock.add(postRequestContent);

			Long tsForProj = largestTimestampReceived_synch_lock.get(postRequestContent.getProjectId());

			if (tsForProj == null) {
				largestTimestampReceived_synch_lock.put(postRequestContent.getProjectId(),
						postRequestContent.getTimestamp());
				tsForProj = postRequestContent.getTimestamp();
			}

			// If we receive an out-of-order timestamp, log a severe error
			if (postRequestContent.getTimestamp() < tsForProj) {
				log.err("SEVERE: An out-of-order timestamp was received in " + this.getClass().getSimpleName()
						+ ", received: " + postRequestContent.getTimestamp() + " largest seen:" + tsForProj);
				severeErrorOccurred_synch_lock = true;
			} else {
				largestTimestampReceived_synch_lock.put(postRequestContent.getProjectId(),
						postRequestContent.getTimestamp());
			}

			// Check if the PRC contains any dupes (which it shouldn't)
			if (!duplicatesDetected_synch_lock) {
				duplicatesDetected_synch_lock = postRequestContent.containsDuplicates();
			}
		}
	}

	public void clear() {
		synchronized (lock) {
			postRequests_synch_lock.clear();
			duplicatesDetected_synch_lock = false;
			severeErrorOccurred_synch_lock = false;
		}
	}

	public List<PostRequestContent> getPostRequests() {
		List<PostRequestContent> result = new ArrayList<>();

		synchronized (lock) {
			if (duplicatesDetected_synch_lock) {
				fail("Duplicates detected");
			}

			if (severeErrorOccurred_synch_lock) {
				fail("Severe error occurred elsewhere.");
			}
			result.addAll(postRequests_synch_lock);
		}

		return result;
	}

	public List<PostRequestContent> getPostRequestsById(ProjectToWatchJson ptw) {
		List<PostRequestContent> result = new ArrayList<>();

		String projectId = ptw.getProjectID();

		synchronized (lock) {
			if (duplicatesDetected_synch_lock) {
				fail("Duplicates detected");
			}

			if (severeErrorOccurred_synch_lock) {
				fail("Severe error occurred elsewhere.");
			}

			result.addAll(postRequests_synch_lock);
		}

		result = result.stream().filter(e -> e.getProjectId().equals(projectId)).collect(Collectors.toList());

		return result;
	}

	public List<ChangedFileEntry> getAllChangedFileEntriesByProjectId(ProjectToWatchJson ptw) {

		List<ChangedFileEntry> result = new ArrayList<>();

		List<PostRequestContent> prcList = getPostRequestsById(ptw);
		prcList.forEach(e -> {
			result.addAll(e.getChangedFileEntries());
		});

		return result;

	}
}
