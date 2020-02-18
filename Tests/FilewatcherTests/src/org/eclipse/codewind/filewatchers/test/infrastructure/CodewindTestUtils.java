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

import java.io.ByteArrayOutputStream;
import java.io.IOException;
import java.io.InputStream;

public class CodewindTestUtils {

	public static String readStringFromStream(InputStream is) throws IOException {

		ByteArrayOutputStream baos = new ByteArrayOutputStream();

		byte[] barr = new byte[1024 * 64];
		int c;
		while (-1 != (c = is.read(barr))) {
			baos.write(barr, 0, c);
		}

		return new String(baos.toByteArray());

	}

	public static void sleep(long timeInMsecs) {
		try {
			Thread.sleep(timeInMsecs);
		} catch (InterruptedException e) {
			throw new RuntimeException(e);
		}
	}

	public static void throwAsUnchecked(Throwable t) {
		if (t instanceof RuntimeException) {
			throw (RuntimeException) t;
		} else if (t instanceof Error) {
			throw (Error) t;
		} else {
			throw new RuntimeException(t);
		}

	}

	/**
	 * Take the OS-specific path format, and convert it to a standardized non-OS
	 * specific format. This method should only effect Windows paths.
	 */
	public static String normalizePath(String pathParam) {
		String absPath = pathParam.replace("\\", "/");

		absPath = convertFromWindowsDriveLetter(absPath);
		absPath = normalizeDriveLetter(absPath);

		return stripTrailingSlash(absPath);

	}

	/** C:\helloThere -> /c/helloThere */
	private static String convertFromWindowsDriveLetter(String absolutePath) {

		if (!isWindowsAbsolutePath(absolutePath)) {
			return absolutePath;
		}

		absolutePath = absolutePath.replace("\\", "/");

		char char0 = absolutePath.charAt(0);

		// Strip first two characters
		absolutePath = absolutePath.substring(2);

		absolutePath = "/" + Character.toLowerCase(char0) + absolutePath;

		return absolutePath;

	}

	/** Ensure that the drive is lowercase for Unix-style paths from Windows. */
	private static String normalizeDriveLetter(String absolutePath) {

		if (absolutePath.contains("\\")) {
			throw new IllegalArgumentException("This function does not support Windows-style paths.");
		}
		if (absolutePath.length() < 2) {
			return absolutePath;
		}

		if (!absolutePath.startsWith("/")) {
			throw new IllegalArgumentException("Path should begin with forward slash: " + absolutePath);
		}

		char char0 = absolutePath.charAt(0);
		char char1 = absolutePath.charAt(1);

		// Special case the absolute path of only 2 characters.
		if (absolutePath.length() == 2) {
			if (char0 == '/' && Character.isLetter(char1) && Character.isUpperCase(char1)) {
				return "/" + Character.toLowerCase(char1);
			} else {
				return absolutePath;
			}

		}

		char char2 = absolutePath.charAt(2);
		if (char0 == '/' && char2 == '/' && Character.isLetter(char1) && Character.isUpperCase(char1)) {

			return "/" + Character.toLowerCase(char1) + char2 + absolutePath.substring(3);

		} else {
			return absolutePath;
		}
	}

	/**
	 * A windows absolute path will begin with a letter followed by a colon: C:\
	 */
	private static boolean isWindowsAbsolutePath(String absolutePath) {
		if (absolutePath.length() < 2) {
			return false;
		}

		char char0 = absolutePath.charAt(0);

		if (!Character.isLetter(char0)) {
			return false;
		}

		if (absolutePath.charAt(1) != ':') {
			return false;
		}

		return true;
	}

	private static String stripTrailingSlash(String str) {

		while (str.endsWith("/")) {

			str = str.substring(0, str.length() - 1);

		}

		return str;

	}

}
