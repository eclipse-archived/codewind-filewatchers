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

import javax.servlet.http.HttpServletRequest;
import javax.servlet.http.HttpServletResponse;

/**
 * Set 'ENABLED' to true in order to run the tests in "chaos mode". In this
 * mode, HTTP requests will be randomly delayed and/or aborted (with error code
 * 500), and WebSocket messages will be delayed and/or dropped (disconnected
 * without writing the WebSocket message.)
 * 
 * This mode exists to test the robustness of filewatchers under extreme
 * conditions (unreliable/slow network conditions), which should ensure that
 * also they perform reliably outside extreme conditions.
 */
public class ChaosEngineering {

	// If enabled, outputs a string whenever chaos ensues.
	public static final boolean OUTPUT_ACTIONS = false;

	public static final boolean ENABLED = false;
	private boolean warningMessagePrinted_synch_lock = false;

	private final Object lock = new Object();

	private final double httpRequestFailPercent = 0.5d;
	private final double httpRequestDelayPercent = 0.5d;
	private final long minimumHttpDelayInMsecs = 1000;
	private final long maxVarianceHttpDelayInMsecs = 2000;

	private final double wsRequestFailPercent = 0.5d;
	private final double wsRequestDelayPercent = 0.5d;
	private final long minimumWsDelayInMsecs = 1000;
	private final long maxVarianceWsDelayInMsecs = 2000;

	private static final CodewindTestLogger log = CodewindTestLogger.getInstance();

	protected ChaosEngineering() {
	}

	/**
	 * Returns true if the response has been failed, or false otherwise
	 * 
	 * @param response
	 * @param request
	 */
	public boolean failOrDelayResponse(HttpServletRequest request, HttpServletResponse response) {
		if (!ENABLED) {
			return false;
		}

		ensureWarningMessagePrinted();

		if (!delayOrHttpRequestFail()) {
			return false;
		}

		log.out("Chaos engineering: Failing HTTP response. " + request.getMethod() + " " + request.getRequestURI());

		response.setStatus(500);

		return true;
	}

	private void ensureWarningMessagePrinted() {
		synchronized (lock) {
			if (warningMessagePrinted_synch_lock) {
				return;
			}
			warningMessagePrinted_synch_lock = true;
		}

		log.err("!!");
		log.err("!!");
		log.err("!!");
		log.err("!! WARNING: Chaos engineering tests are enabled, which may effect test results in unexpected ways.");
		log.err("!!");
		log.err("!!");
		log.err("!!");
	}

	private boolean delayOrHttpRequestFail() {

		if (Math.random() < httpRequestDelayPercent) {
			long sleepTime = (long) (Math.random() * (double) maxVarianceHttpDelayInMsecs) + minimumHttpDelayInMsecs;
			if (OUTPUT_ACTIONS) {
				log.out("Chaos engineering: Delaying HTTP response for " + sleepTime + " msecs. ");
			}
			CodewindTestUtils.sleep(sleepTime);
		}

		return Math.random() < httpRequestFailPercent;

	}

	public boolean failOrDelayWebSocket() {
		if (!ENABLED) {
			return false;
		}

		ensureWarningMessagePrinted();

		return delayOrWsRequestFail();

	}

	private boolean delayOrWsRequestFail() {

		if (Math.random() < wsRequestDelayPercent) {
			long sleepTime = (long) (Math.random() * (double) maxVarianceWsDelayInMsecs) + minimumWsDelayInMsecs;
			if (OUTPUT_ACTIONS) {
				log.out("Chaos engineering: Delaying WebSocket write for " + sleepTime + " msecs. ");
			}
			CodewindTestUtils.sleep(sleepTime);
		}

		return Math.random() < wsRequestFailPercent;

	}

}
