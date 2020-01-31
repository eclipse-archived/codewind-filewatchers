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

package org.eclipse.codewind.mockcwctl.json;

import java.util.ArrayList;
import java.util.List;

public class MockProjectWatchJson {

	List<String> ignoredPaths = new ArrayList<>();
	List<String> ignoredFilenames = new ArrayList<>();
	List<String> filesToWatch = new ArrayList<>();

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

	public List<String> getFilesToWatch() {
		return filesToWatch;
	}

	public void setFilesToWatch(List<String> filesToWatch) {
		this.filesToWatch = filesToWatch;
	}

}
