/*******************************************************************************
 * Copyright (c) 2019 IBM Corporation
 * All rights reserved. This program and the accompanying materials
 * are made available under the terms of the Eclipse Public License v2.0
 * which accompanies this distribution, and is available at
 * http://www.eclipse.org/legal/epl-v20.html
 *
 * Contributors:
 *     IBM Corporation - initial API and implementation
 *******************************************************************************/

package org.eclipse.codewind.filewatchers.core.internal;

import static org.junit.Assert.assertFalse;
import static org.junit.Assert.assertTrue;

import org.eclipse.codewind.filewatchers.core.PathFilter;
import org.eclipse.codewind.filewatchers.core.ProjectToWatch;
import org.junit.Test;

/** Unit test for PathFilter. */
public class PathFilterTest {

	private static final String PROJECT_ID = "project-id";

	/**
	 * Rule: Ensure we are using a black list for paths and filenames
	 */
	@Test
	public void testFoundationRule1() {
		ProjectToWatch ptw = new ProjectToWatch(PROJECT_ID, "/C/files");

		PathFilter pf = new PathFilter(ptw);

		assertFalse(pf.isFilteredOutByPath("/any/dir/filename"));
		assertFalse(pf.isFilteredOutByFilename("/any/dir/filename"));
	}

	/**
	 * Rule: Path ignore of /target/* should match /target/stuff
	 */
	@Test
	public void testRule1() {

		ProjectToWatch ptw = new ProjectToWatch(PROJECT_ID, "/C/files");

		ptw.getIgnoredPaths().add("/target/*");

		PathFilter pf = new PathFilter(ptw);

		assertTrue(pf.isFilteredOutByPath("/target/stuff"));

	}

	/**
	 * Rule: Path ignore of /target should not match /target/stuff
	 */
	@Test
	public void testRule2() {

		ProjectToWatch ptw = new ProjectToWatch(PROJECT_ID, "/C/files");

		ptw.getIgnoredPaths().add("/target");

		PathFilter pf = new PathFilter(ptw);

		assertFalse(pf.isFilteredOutByPath("/target/stuff"));

	}

	/**
	 * Rule: Ignore filename of *.class should not affect filtered out by path
	 */
	@Test
	public void testRule3() {
		ProjectToWatch ptw = new ProjectToWatch(PROJECT_ID, "/C/files");

		ptw.getIgnoredFilenames().add("*.class");

		PathFilter pf = new PathFilter(ptw);

		// Note this is filter by path, not by filename
		assertFalse(pf.isFilteredOutByPath("/dir/dir/file.class"));

	}

	/**
	 * Rule: Ignore filename of *.class, should match file.class
	 */
	@Test
	public void testRule4() {
		ProjectToWatch ptw = new ProjectToWatch(PROJECT_ID, "/C/files");

		ptw.getIgnoredFilenames().add("*.class");

		PathFilter pf = new PathFilter(ptw);

		assertTrue(pf.isFilteredOutByFilename("/dir/dir/file.class"));

	}

	// Rule: Ignore filename of dir3 should match a trailing directory named dir3.
	// Rule: Ignore filename of dir3 should not match an inner directory named dir3.
	@Test
	public void testRule5() {
		ProjectToWatch ptw = new ProjectToWatch(PROJECT_ID, "/C/files");

		ptw.getIgnoredFilenames().add("dir3");

		PathFilter pf = new PathFilter(ptw);

		assertTrue(pf.isFilteredOutByFilename("/dir/dir2/dir3"));

		assertTrue(pf.isFilteredOutByFilename("/dir/dir3/dir2"));

		assertFalse(pf.isFilteredOutByFilename("/dir/dir4/dir2"));

	}

	@Test
	public void testRule6() {
		ProjectToWatch ptw = new ProjectToWatch(PROJECT_ID, "/C/files");

		ptw.getIgnoredFilenames().add("*dir*");

		PathFilter pf = new PathFilter(ptw);

		assertTrue(pf.isFilteredOutByFilename("/dir/dir2/dir3"));

	}

	@Test
	public void testRule7() {
		ProjectToWatch ptw = new ProjectToWatch(PROJECT_ID, "/C/files");

		ptw.getIgnoredFilenames().add("*dir");

		PathFilter pf = new PathFilter(ptw);

		assertTrue(pf.isFilteredOutByFilename("/dir/dir2/dir3"));

	}

	@Test
	public void testRule8() {
		ProjectToWatch ptw = new ProjectToWatch(PROJECT_ID, "/C/files");

		ptw.getIgnoredPaths().add("*/dir2/*");

		PathFilter pf = new PathFilter(ptw);

		assertTrue(pf.isFilteredOutByPath("/dir/dir2/dir3"));

	}

	@Test
	public void testRule9() {
		ProjectToWatch ptw = new ProjectToWatch(PROJECT_ID, "/C/files");

		ptw.getIgnoredFilenames().add("inner");

		PathFilter pf = new PathFilter(ptw);

		assertTrue(pf.isFilteredOutByFilename("/dir/inner/dir2"));

	}

	@Test
	public void testRule10() {
		ProjectToWatch ptw = new ProjectToWatch(PROJECT_ID, "/C/files");

		ptw.getIgnoredPaths().add("/target");

		PathFilter pf = new PathFilter(ptw);

		assertFalse(pf.isFilteredOutByPath("/dir/target/dir2"));

	}

}
