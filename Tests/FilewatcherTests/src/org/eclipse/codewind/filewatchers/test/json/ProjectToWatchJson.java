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

package org.eclipse.codewind.filewatchers.test.json;

import java.io.File;
import java.util.ArrayList;
import java.util.Collections;
import java.util.List;
import java.util.UUID;

import org.eclipse.codewind.filewatchers.test.infrastructure.CodewindTestUtils;

import com.fasterxml.jackson.annotation.JsonIgnore;

public class ProjectToWatchJson {

	private String projectID;

	/** Path (inside the container) to monitor */
	private String pathToMonitor;

	private List<String> ignoredPaths = new ArrayList<>();
	private List<String> ignoredFilenames = new ArrayList<>();

	private String changeType;

	private String projectWatchStateId;

	private String type;

	private Long projectCreationTime = null;

	private List<RefPathEntry> refPaths = new ArrayList<>();

	public ProjectToWatchJson() {
	}

	@JsonIgnore
	private File localPathToMonitor;

	public String getProjectID() {
		return projectID;
	}

	public void setProjectID(String projectID) {
		this.projectID = projectID;
	}

	public String getPathToMonitor() {
		return pathToMonitor;
	}

	public void setPathToMonitor(String pathToMonitor) {
		this.pathToMonitor = pathToMonitor;
	}

	public List<String> getIgnoredPaths() {
		return ignoredPaths;
	}

	public void setIgnoredPaths(List<String> ignoredPaths) {
		this.ignoredPaths = ignoredPaths;
	}

	public List<String> getIgnoredFilenames() {
		return ignoredFilenames;
	}

	public void setIgnoredFilenames(List<String> ignoredFilenames) {
		this.ignoredFilenames = ignoredFilenames;
	}

	public String getChangeType() {
		return changeType;
	}

	public void setChangeType(String changeType) {
		this.changeType = changeType;
	}

	public ProjectToWatchJson clone() {
		ProjectToWatchJson result = new ProjectToWatchJson();
		result.setChangeType(this.getChangeType());
		result.setIgnoredFilenames(new ArrayList<>(this.getIgnoredFilenames()));
		result.setIgnoredPaths(new ArrayList<>(this.getIgnoredPaths()));
		result.setPathToMonitor(this.getPathToMonitor());
		result.setProjectID(this.getProjectID());
		result.setProjectWatchStateId(this.getProjectWatchStateId());
		result.setType(this.getType());
		result.setProjectCreationTime(this.getProjectCreationTime());
		result.setRefPaths(new ArrayList<>(this.getRefPaths() != null ? this.getRefPaths() : Collections.emptyList()));
		return result;
	}

	public String getType() {
		return type;
	}

	public void setType(String type) {
		this.type = type;
	}

	@JsonIgnore
	public File getLocalPathToMonitor() {
		return localPathToMonitor;
	}

	@JsonIgnore
	public void setLocalPathToMonitor(File localPathToMonitor) {
		this.localPathToMonitor = localPathToMonitor;
	}

	public String getProjectWatchStateId() {
		return projectWatchStateId;
	}

	public void setProjectWatchStateId(String projectWatchStateId) {
		this.projectWatchStateId = projectWatchStateId;
	}

	public Long getProjectCreationTime() {
		return projectCreationTime;
	}

	public void setProjectCreationTime(Long projectCreationTime) {
		this.projectCreationTime = projectCreationTime;
	}

	@JsonIgnore
	public void regenerateWatchId() {
		this.projectWatchStateId = UUID.randomUUID().toString();
	}

	public List<RefPathEntry> getRefPaths() {
		return refPaths;
	}

	public void setRefPaths(List<RefPathEntry> refPaths) {
		this.refPaths = refPaths;
	}

	public static class RefPathEntry {

		String from;
		String to;

		public RefPathEntry() {
		}

		public RefPathEntry(String from, String to) {
			this.from = CodewindTestUtils.normalizePath(from);
			this.to = to;
		}

		public String getFrom() {
			return from;
		}

		public void setFrom(String from) {
			this.from = from;
		}

		public String getTo() {
			return to;
		}

		public void setTo(String to) {
			this.to = to;
		}

	}
}
