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

import static org.junit.Assert.assertNotNull;
import static org.junit.Assert.assertTrue;
import static org.junit.Assert.fail;

import java.util.ArrayList;
import java.util.Arrays;
import java.util.List;

import org.eclipse.codewind.filewatchers.core.WatchEventEntry.EventType;
import org.eclipse.codewind.filewatchers.core.internal.FileChangeEventBatchUtil.ChangedFileEntry;
import org.junit.Test;

public class FileChangeEventBatchUtilTest {

	@Test
	public void testSimple() {

		long timestamp = 1;

		for (EventType e : Arrays.asList(EventType.CREATE, EventType.DELETE)) {

			List<ChangedFileEntry> cfe = new ArrayList<>();

			cfe.add(new ChangedFileEntry("/2", false, EventType.MODIFY, timestamp++));
			cfe.add(new ChangedFileEntry("/1", false, e, timestamp++));
			cfe.add(new ChangedFileEntry("/2", false, EventType.MODIFY, timestamp++));
			cfe.add(new ChangedFileEntry("/1", false, e, timestamp++));
			cfe.add(new ChangedFileEntry("/1", false, EventType.MODIFY, timestamp++));

			cfe.add(new ChangedFileEntry("/1", false, EventType.MODIFY, timestamp++));

			FileChangeEventBatchUtil.removeDuplicateEventsOfType(cfe, e);

			assertAscendingTimestamp(cfe);

			assertTrue(countByType(cfe, e) == 1);

			assertTrue(countByType(cfe, EventType.MODIFY) == 4);

		}

	}

	@Test
	public void testDontRemoveIfNotContiguous() {

		long timestamp = 1;

		for (EventType e : Arrays.asList(EventType.CREATE, EventType.DELETE)) {

			List<ChangedFileEntry> cfe = new ArrayList<>();

			cfe.add(new ChangedFileEntry("/1", false, e, timestamp++));
			cfe.add(new ChangedFileEntry("/1", false, anotherEvent(e), timestamp++));
			cfe.add(new ChangedFileEntry("/1", false, e, timestamp++));
			cfe.add(new ChangedFileEntry("/1", false, anotherEvent(e), timestamp++));
			cfe.add(new ChangedFileEntry("/1", false, e, timestamp++));

			FileChangeEventBatchUtil.removeDuplicateEventsOfType(cfe, e);
			assertAscendingTimestamp(cfe);

			assertTrue(countByType(cfe, e) == 3);

			assertTrue(cfe.size() == 5);

		}
	}

	private static EventType anotherEvent(EventType type) {
		EventType result = Arrays.asList(EventType.values()).stream().filter(e -> e != type).findFirst().orElse(null);

		assertNotNull(result);

		return result;
	}

	private static long countByType(List<ChangedFileEntry> entries, EventType type) {
		return entries.stream().filter(e -> e.getType() == type).count();
	}

	private static void assertAscendingTimestamp(List<ChangedFileEntry> entries) {

		long lastTimestamp = -1;
		for (ChangedFileEntry cfe : entries) {

			if (cfe.getTimestamp() < lastTimestamp) {
				fail("Timestamp is not ascending: " + entries);
			}

			lastTimestamp = cfe.getTimestamp();

		}

	}
}
