/*******************************************************************************
 * Copyright (c) 2019, 2020 IBM Corporation
 * All rights reserved. This program and the accompanying materials
 * are made available under the terms of the Eclipse Public License v2.0
 * which accompanies this distribution, and is available at
 * http://www.eclipse.org/legal/epl-v20.html
 *
 * Contributors:
 *     IBM Corporation - initial API and implementation
 *******************************************************************************/

package org.eclipse.codewind.filewatchers.test.json;

import java.util.ArrayList;
import java.util.List;

public class ProjectToWatchListJson {

	private List<ProjectToWatchJson> projects = new ArrayList<>();

	public List<ProjectToWatchJson> getProjects() {
		return projects;
	}

	public void setProjects(List<ProjectToWatchJson> projects) {
		this.projects = projects;
	}

}
