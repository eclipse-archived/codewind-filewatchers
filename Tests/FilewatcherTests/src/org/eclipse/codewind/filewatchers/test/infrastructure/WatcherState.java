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

import static org.junit.Assert.assertNotNull;

import java.util.ArrayList;
import java.util.Arrays;
import java.util.Collections;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

import org.eclipse.codewind.filewatchers.test.json.ProjectToWatchJson;
import org.eclipse.codewind.filewatchers.test.json.WatchChangeJson;

import com.fasterxml.jackson.annotation.JsonInclude.Include;
import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.databind.ObjectMapper;

public class WatcherState {

	private final Map<String /* projectId */, ProjectToWatchJson> projectsToWatch_synch_lock = new HashMap<>();

	private final Map<String /* projectId */, Map<String /* watch status id */, Boolean>> watchStatusReceived_synch_lock = new HashMap<>();

	private final Object lock = new Object();

	public WatcherState() {
	}

	public List<ProjectToWatchJson> getProjects() {
		List<ProjectToWatchJson> result = new ArrayList<>();
		synchronized (lock) {
			result.addAll(projectsToWatch_synch_lock.values());
		}

		return result;
	}

	public void addWatchStatus(String projectId, String projectWatchStatusId, boolean success) {

		synchronized (lock) {
			Map<String, Boolean> watchMap = watchStatusReceived_synch_lock.computeIfAbsent(projectId,
					e -> new HashMap<>());

			Boolean b = watchMap.get(projectWatchStatusId);
			if (b != null && b != success) {
				throw new IllegalArgumentException("Watch status changed from " + b + " to " + success + " for "
						+ projectId + " " + projectWatchStatusId);
			}

			watchMap.put(projectWatchStatusId, success);
		}

	}

	/*
	 * Returns true or false if a watch status has been received, or null otherwise.
	 */
	public Boolean getWatchStatus(String projectId, String projectWatchStatusId) {
		synchronized (lock) {
			Map<String, Boolean> m = watchStatusReceived_synch_lock.get(projectId);
			if (m == null) {
				return null;
			}

			return m.get(projectWatchStatusId);

		}

	}

	public void addOrUpdateProject(ProjectToWatchJson project) {

		this.addOrUpdateProjects(Arrays.asList(new ProjectToWatchJson[] { project }));
	}

	public void addOrUpdateProjects(List<ProjectToWatchJson> projects) {

		List<ProjectToWatchJson> added = new ArrayList<>();
		List<ProjectToWatchJson> changed = new ArrayList<>();

		synchronized (lock) {

			for (ProjectToWatchJson project : projects) {

				ProjectToWatchJson match = projectsToWatch_synch_lock.values().stream()
						.filter(e -> e.getProjectID().equals(project.getProjectID())).findAny().orElse(null);

				if (match != null) {
					projectsToWatch_synch_lock.put(project.getProjectID(), project.clone());
					changed.add(project.clone());

				} else {
					projectsToWatch_synch_lock.put(project.getProjectID(), project.clone());
					added.add(project.clone());
				}

			}

		}

		communicateChangedProjects(changed, added, Collections.emptyList());
	}

	public void clearAllProjects() {
		synchronized (lock) {
			List<ProjectToWatchJson> proj = getProjects();

			if (proj.size() == 0) {
				return;
			}

			deleteProjects(proj);
		}
	}

	public void deleteProjects(List<ProjectToWatchJson> projects) {

		synchronized (lock) {
			for (ProjectToWatchJson project : projects) {
				ProjectToWatchJson removed = projectsToWatch_synch_lock.remove(project.getProjectID());
				assertNotNull("deleteProjects called with project that could not be found in list: "
						+ project.getProjectID() + " " + project.getPathToMonitor(), removed);
			}
		}

		communicateChangedProjects(Collections.emptyList(), Collections.emptyList(), projects);
	}

	private void communicateChangedProjects(List<ProjectToWatchJson> changedProjects,
			List<ProjectToWatchJson> addedProjects, List<ProjectToWatchJson> deletedProjects) {

		WatchChangeJson result = new WatchChangeJson();

		result.setType("watchChanged");

		for (ProjectToWatchJson j : changedProjects) {
			j.setChangeType("update");
			result.getProjects().add(j);
		}

		for (ProjectToWatchJson j : addedProjects) {
			j.setChangeType("add");
			result.getProjects().add(j);
		}

		for (ProjectToWatchJson k : deletedProjects) {
			ProjectToWatchJson j = new ProjectToWatchJson();
			j.setIgnoredFilenames(null);
			j.setIgnoredPaths(null);
			j.setChangeType("delete");
			j.setPathToMonitor(null);
			j.setProjectID(k.getProjectID());
			j.setProjectWatchStateId(null);
			j.setProjectCreationTime(null);
			result.getProjects().add(j);
		}

		// Remove ignoredFilenames from the JSON if it is empty. This simulates the
		// codewind server behaviour, where ignoredFilename is not currently used.
		Arrays.asList(changedProjects, addedProjects, deletedProjects).stream().flatMap(e -> e.stream())
				.filter(e -> e.getIgnoredFilenames().size() == 0).forEach(e -> {
					e.setIgnoredFilenames(null);
				});

		ObjectMapper om = new ObjectMapper();
		om.setSerializationInclusion(Include.NON_NULL);

		String msg;
		try {
			msg = om.writeValueAsString(result);
		} catch (JsonProcessingException e1) {
			e1.printStackTrace();
			return;
		}

		CodewindTestState.getInstance().getConnectionState().ifPresent(e -> {
			e.writeToActiveSessions(msg, true);
		});

	}

}
