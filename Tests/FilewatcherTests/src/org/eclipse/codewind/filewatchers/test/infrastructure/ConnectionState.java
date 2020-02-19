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

import java.util.ArrayList;
import java.util.Iterator;
import java.util.List;
import java.util.concurrent.atomic.AtomicInteger;

import org.eclipse.codewind.filewatchers.test.json.DebugMessageOverWebSocketJson;
import org.eclipse.jetty.websocket.api.RemoteEndpoint;
import org.eclipse.jetty.websocket.api.Session;

import com.fasterxml.jackson.annotation.JsonInclude.Include;
import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.databind.ObjectMapper;

public class ConnectionState {

	private final Object lock = new Object();

	private final List<Session> connectedWebSockets_synch_lock = new ArrayList<>();

	private boolean disposed_synch_lock = false;

	private final boolean filterOutWebSocketMessages;

	private static final CodewindTestLogger log = CodewindTestLogger.getInstance();

	public ConnectionState(boolean filterOutWebSocketMessages) {
		this.filterOutWebSocketMessages = filterOutWebSocketMessages;
	}

	public void writeToActiveSessions(String msg, boolean echoString/* , boolean blockingWrite */) {

		List<Session> sessions = new ArrayList<>();
		synchronized (lock) {
			if (disposed_synch_lock) {
				return;
			}
			sessions.addAll(connectedWebSockets_synch_lock);
		}

		AtomicInteger count_synch = new AtomicInteger(0);

		// Filter out non-debug WebSocket messages, if enabled.
		if (filterOutWebSocketMessages && !msg.contains("debug")) {
			log.out("ignoring: " + msg);
			return;
		}

		if ((msg == null || !msg.contains("debug"))
				&& CodewindTestState.getInstance().getChaosEngineering().failOrDelayWebSocket()) {
			log.out("Chaos engineering: disconnecting all WebSocket connections.");
			// As per the docs, 'disconnect()' will issue a harsh disconnect of the
			// underlying connection. This will terminate the connection, without sending a
			// WebSocket close frame. (eg this is different from e.close() ).
			sessions.forEach(e -> {
				try {
					e.disconnect();
				} catch (Exception ex) {
					/* ignore */ }
			});
			return;
		}

		sessions.forEach(e -> {
			new Thread() {
				public void run() {

//					String thing = "!";
//					while (thing.length() < 256 * 1024) {
//						thing = thing + thing;
//					}

					try {
						RemoteEndpoint re = e.getRemote();
						// Synchronize on the session before we write, as Jetty WebSocket client does
						// not support multiple simultaneous writes to 're' from multiple threads.
						// https://stackoverflow.com/questions/26264508/websocket-async-send-can-result-in-blocked-send-once-queue-filled
						synchronized (e) {
//							re.sendString(thing);
							re.sendString(msg);
						}
						if (echoString) {
							log.out("Sent string: " + msg);
						}
					} catch (Exception ex) {
						/* ignore */
					} finally {
						synchronized (count_synch) {
							count_synch.incrementAndGet();
							count_synch.notify();
						}
					}
				};
			}.start();
		});

//		if (blockingWrite) {
//			long expireTimeInNanos = System.nanoTime() + TimeUnit.NANOSECONDS.convert(30, TimeUnit.SECONDS);
//
//			while (System.nanoTime() < expireTimeInNanos) {
//				synchronized (count_synch) {
//					if (count_synch.get() == sessions.size()) {
//						return;
//					} else {
//						try {
//							count_synch.wait();
//						} catch (InterruptedException e1) {
//							throw new RuntimeException(e1);
//						}
//					}
//				}
//			}
//		}
	}

	public void addSession(Session s) {
		synchronized (lock) {
			if (disposed_synch_lock) {

				new Thread() {
					public void run() {
						log.out("Ignored session: " + s);
						s.close();
					}
				}.start();

				return;
			}

			this.connectedWebSockets_synch_lock.add(s);
			lock.notify();

		}

		log.out("Added session: " + s);

	}

	public void removeSession(Session s) {
		synchronized (lock) {

			for (Iterator<Session> it = connectedWebSockets_synch_lock.iterator(); it.hasNext();) {
				Session curr = it.next();

				if (curr == s) {
					log.out("Removed session: " + s);
					it.remove();
				}

			}

		}
	}

	public void waitForAtLeastOneSession() {
		synchronized (lock) {
			while (connectedWebSockets_synch_lock.size() == 0) {
				try {
					lock.wait(100);
				} catch (InterruptedException e) {
					throw new RuntimeException(e);
				}
			}

		}
	}

	public void dispose() {

		synchronized (lock) {

			if (disposed_synch_lock) {
				return;
			}

			this.disposed_synch_lock = true;

			connectedWebSockets_synch_lock.forEach(e -> {

				new Thread() {
					public void run() {
						log.out("Closed session: " + e);
						e.close();
					}
				}.start();

			});

			connectedWebSockets_synch_lock.clear();

		}

	}

	public void sendDebugMessage(String debugMessage) {
		DebugMessageOverWebSocketJson dmowj = new DebugMessageOverWebSocketJson();
		dmowj.setType("debug");
		dmowj.setMsg(debugMessage);

		ObjectMapper om = new ObjectMapper();
		om.setSerializationInclusion(Include.NON_NULL);

		String msg;
		try {
			msg = om.writeValueAsString(dmowj);
		} catch (JsonProcessingException e1) {
			throw new RuntimeException(e1);
		}

		writeToActiveSessions(msg, false);

	}

}
