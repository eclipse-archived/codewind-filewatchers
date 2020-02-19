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

import org.eclipse.jetty.websocket.api.Session;
import org.eclipse.jetty.websocket.api.WebSocketAdapter;
import org.eclipse.jetty.websocket.servlet.WebSocketServlet;
import org.eclipse.jetty.websocket.servlet.WebSocketServletFactory;

@SuppressWarnings("serial")
public class CodewindTestWebSocketServlet extends WebSocketServlet {

	private static final CodewindTestLogger log = CodewindTestLogger.getInstance();

	public CodewindTestWebSocketServlet() {
	}

	@Override
	public void configure(WebSocketServletFactory factory) {
		factory.register(CodeWindTestWebSocketAdapter.class);
	}

	public static class CodeWindTestWebSocketAdapter extends WebSocketAdapter {

		public CodeWindTestWebSocketAdapter() {
		}

		@Override
		public void onWebSocketConnect(Session sess) {
			log.out("WebSocket connected.");
			CodewindTestState.getInstance().getConnectionState().ifPresent(e -> {
				e.addSession(sess);
			});
		}

		@Override
		public void onWebSocketClose(int statusCode, String reason) {
			log.out("WebSocket closed.");

			final Session s = getSession();
			CodewindTestState.getInstance().getConnectionState().ifPresent(e -> {
				e.removeSession(s);
			});

		}

		@Override
		public void onWebSocketText(String message) {
			/* ignore */
		}
	}

}
