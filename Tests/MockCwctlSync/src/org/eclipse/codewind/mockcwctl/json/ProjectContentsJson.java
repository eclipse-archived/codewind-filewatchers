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

public class ProjectContentsJson {

	List<FileEntry> entries = new ArrayList<>();

	public List<FileEntry> getEntries() {
		return entries;
	}

	public void setEntries(List<FileEntry> entries) {
		this.entries = entries;
	}

	public static class FileEntry {

		boolean directory;
		String path;
		long modification;

		public boolean isDirectory() {
			return directory;
		}

		public void setDirectory(boolean directory) {
			this.directory = directory;
		}

		public String getPath() {
			return path;
		}

		public void setPath(String path) {
			this.path = path;
		}

		public long getModification() {
			return modification;
		}

		public void setModification(long modification) {
			this.modification = modification;
		}

	}
}
