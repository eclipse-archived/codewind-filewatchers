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

import static org.junit.Assert.assertTrue;
import static org.junit.Assert.fail;

import java.nio.file.Paths;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.List;

import org.eclipse.codewind.filewatchers.core.PathUtils;
import org.eclipse.codewind.filewatchers.core.WatchEventEntry;
import org.eclipse.codewind.filewatchers.core.WatchEventEntry.EventType;
import org.junit.Test;

/** Unit test for PathUtils. */
public class PathUtilsTest {

	@Test
	public void testGetAbsolutePathWithUnixSeparators() {

		List<String[]> toCompare = new ArrayList<>();
		toCompare.add(new String[] { "c:\\hi\\file", "/c/hi/file" });
		toCompare.add(new String[] { "c:\\hi\\", "/c/hi" });
		toCompare.add(new String[] { "c:\\hi", "/c/hi" });
		toCompare.add(new String[] { "/hi", "/hi" });
		toCompare.add(new String[] { "/hi", "/hi" });
		toCompare.add(new String[] { "/c/hi", "/c/hi" });

		for (String[] entry : toCompare) {
			WatchEventEntry wee = new WatchEventEntry(EventType.CREATE, Paths.get(entry[0]), true);

			String val = wee.getAbsolutePathWithUnixSeparators();
			assertTrue("Input: '" + entry[0] + "'  Expected: '" + entry[1] + "'  Actual: '" + val + "'",
					val.equals(entry[1]));

		}
	}

	@Test
	public void testNormalizeDriveLetter() {
		try {
			PathUtils.normalizeDriveLetter("c:\\thing\\thing2");
			fail("Function should not accept Windows-style paths.");
		} catch (Exception e) {
			// Pass
		}

	}

	@Test
	public void testNormalizeDriveLetter2() {

		List<String[]> toCompare = new ArrayList<>();

		toCompare.add(new String[] { "/home/user", "/home/user" });
		toCompare.add(new String[] { "/C/somepath", "/c/somepath" });
		toCompare.add(new String[] { "/TW/somepath", "/TW/somepath" });
		toCompare.add(new String[] { "/tw/somepath", "/tw/somepath" });
		toCompare.add(new String[] { "/c/somepath", "/c/somepath" });
		toCompare.add(new String[] { "/", "/" });
		toCompare.add(new String[] { "/c", "/c" });
		toCompare.add(new String[] { "/C", "/c" });

		for (String[] entry : toCompare) {

			String val = PathUtils.normalizeDriveLetter(entry[0]);

			assertTrue("Input: '" + entry[0] + "'  Expected: '" + entry[1] + "'  Actual: '" + val + "'",
					val.equals(entry[1]));

		}

		try {
			PathUtils.normalizeDriveLetter("c/somepath");
			fail("An exception should be thrown if the path does not begin with forward slash.");
		} catch (Exception e) {
			// Pass
		}
	}

	@Test
	public void testConvertAbsoluteToLocalOnWindows() {
		List<String[]> toCompare = new ArrayList<>();

		toCompare.add(new String[] { "/c", "c:\\" });
		toCompare.add(new String[] { "/z", "z:\\" });
		toCompare.add(new String[] { "/c/users", "c:\\users" });
		toCompare.add(new String[] { "/z/users", "z:\\users" });

		toCompare.add(new String[] { "/c/users/thing", "c:\\users\\thing" });

		for (String[] entry : toCompare) {

			String val = PathUtils.convertAbsoluteUnixStyleNormalizedPathToLocalFile(entry[0], true);

			assertTrue("Input: '" + entry[0] + "'  Expected: '" + entry[1] + "'  Actual: '" + val + "'",
					val.equals(entry[1]));

		}
	}

	@Test
	public void convertAbsoluteToLocal_testExpectedExceptionsOnWindows() {
		List<String> toTest = Arrays.asList("c/", "/", "/cc");

		for (String entry : toTest) {

			try {
				PathUtils.convertAbsoluteUnixStyleNormalizedPathToLocalFile(entry, true);
				fail("Value should have thrown an exception: " + entry);
			} catch (Exception e) {
				// Pass
			}

		}

	}

	@Test
	public void testConvertAbsoluteToLocalOnUnix() {
		List<String[]> toCompare = new ArrayList<>();

		toCompare.add(new String[] { "/c", "/c" });
		toCompare.add(new String[] { "/c/users", "/c/users" });
		toCompare.add(new String[] { "/c/users/dir", "/c/users/dir" });
		toCompare.add(new String[] { "/", "/" });

		for (String[] entry : toCompare) {

			String val = PathUtils.convertAbsoluteUnixStyleNormalizedPathToLocalFile(entry[0], false);

			assertTrue("Input: '" + entry[0] + "'  Expected: '" + entry[1] + "'  Actual: '" + val + "'",
					val.equals(entry[1]));

		}
	}

	@Test
	public void testSplitRelativeProjectPathIntoComponentPaths() {

		assertTrue(
				PathUtils.splitRelativeProjectPathIntoComponentPaths("/moo/cow").toString().equals("[/moo/cow, /moo]"));

		assertTrue(PathUtils.splitRelativeProjectPathIntoComponentPaths("/moo").toString().equals("[/moo]"));

		assertTrue(PathUtils.splitRelativeProjectPathIntoComponentPaths("/").toString().equals("[]"));

	}

}
