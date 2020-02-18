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

import java.util.Optional;

public class CodewindTestState {

	private static final CodewindTestState instance = new CodewindTestState();

	private CodewindTestState() {
	}

	public static CodewindTestState getInstance() {
		return instance;
	}

	// --------------------------

	private final Object lock = new Object();

	private ConnectionState state_synch_lock = null;

	private final WatcherState watcherState_synch_lock = new WatcherState();

	private final PostRequestListFromWatcher changeListFromWatcher_synch_lock = new PostRequestListFromWatcher();

	private final ChaosEngineering chaosEngineering = new ChaosEngineering();

	public void setConnectionState(ConnectionState state) {
		synchronized (lock) {

			state_synch_lock = state;

		}
	}

	public Optional<ConnectionState> getConnectionState() {

		synchronized (lock) {
			return Optional.ofNullable(state_synch_lock);
		}

	}

	public WatcherState getWatcherState() {
		synchronized (lock) {
			return watcherState_synch_lock;
		}
	}

	public PostRequestListFromWatcher getChangeListFromWatcher() {
		synchronized (lock) {
			return changeListFromWatcher_synch_lock;
		}
	}

	public ChaosEngineering getChaosEngineering() {
		return chaosEngineering;
	}

}
