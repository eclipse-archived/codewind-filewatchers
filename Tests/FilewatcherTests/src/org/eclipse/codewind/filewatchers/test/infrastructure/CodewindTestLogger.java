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

import java.text.SimpleDateFormat;
import java.util.Date;

public class CodewindTestLogger {

	private static final CodewindTestLogger instance = new CodewindTestLogger();

	private static final SimpleDateFormat PRETTY_DATE_FORMAT = new SimpleDateFormat("MMM d h:mm:ss.SSS a");

	private CodewindTestLogger() {
	}

	public static CodewindTestLogger getInstance() {
		return instance;
	}

	public void err(String str) {
		System.err.println(time() + " " + str);
	}

	public void out(String str) {
		System.out.println(time() + " " + str);
	}

	public void out() {
		out("");
	}

	private final String time() {

		return "[" + PRETTY_DATE_FORMAT.format(new Date()) + "]";

	}

}
