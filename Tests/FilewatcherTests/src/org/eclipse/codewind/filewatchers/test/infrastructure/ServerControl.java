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

import org.eclipse.jetty.server.Connector;
import org.eclipse.jetty.server.HttpConfiguration;
import org.eclipse.jetty.server.HttpConnectionFactory;
import org.eclipse.jetty.server.SecureRequestCustomizer;
import org.eclipse.jetty.server.Server;
import org.eclipse.jetty.server.ServerConnector;
import org.eclipse.jetty.server.SslConnectionFactory;
import org.eclipse.jetty.servlet.ServletHandler;
import org.eclipse.jetty.util.ssl.SslContextFactory;

public class ServerControl {

	private Server server = null;

	private ConnectionState connectionState = null;

	private boolean filterOutWebSocketMessages = false;

	public ServerControl() {
	}

	public void startServer() {

		final boolean LISTEN_ON_TLS = false;
		if (LISTEN_ON_TLS) {
			server = new Server();

			HttpConfiguration https = new HttpConfiguration();

			https.addCustomizer(new SecureRequestCustomizer());

			SslContextFactory.Server sslContextFactory = new SslContextFactory.Server();

			sslContextFactory.setTrustAll(true);

			sslContextFactory.setKeyStorePath("C:\\ibm-java8-jdk-805-15\\bin\\keystore");
			sslContextFactory.setKeyStorePassword("password");
			ServerConnector sslConnector = new ServerConnector(server,
					new SslConnectionFactory(sslContextFactory, "http/1.1"), new HttpConnectionFactory(https));

			sslConnector.setPort(9090);

			server.setConnectors(new Connector[] { sslConnector });

		} else {
			server = new Server(9090);
		}

		connectionState = new ConnectionState(filterOutWebSocketMessages);
		CodewindTestState.getInstance().setConnectionState(connectionState);

		ServletHandler handler = new ServletHandler();
		server.setHandler(handler);
		handler.addServletWithMapping(CodewindTestApiServlet.class, "/api/v1/projects/*");

		handler.addServletWithMapping(CodewindTestWebSocketServlet.class, "/websockets/file-changes/v1");

		try {
			server.start();
		} catch (Exception e) {
			throw new RuntimeException(e);
		}

	}

	public void setFilterOutWebSocketMessages(boolean filterOutWebSocketMessages) {
		this.filterOutWebSocketMessages = filterOutWebSocketMessages;
	}

	public void waitForConnectedWebSocket() {
		connectionState.waitForAtLeastOneSession();
	}

	public void stopServer() {
		if (server == null) {
			return;
		}

		try {
			server.stop();
		} catch (Exception e) {
			throw new RuntimeException(e);
		}

		connectionState = null;
		server = null;
	}

}
