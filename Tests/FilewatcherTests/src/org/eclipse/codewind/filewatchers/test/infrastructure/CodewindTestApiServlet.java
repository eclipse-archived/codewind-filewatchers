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

import java.io.ByteArrayInputStream;
import java.io.IOException;
import java.util.Arrays;
import java.util.Base64;
import java.util.List;
import java.util.stream.Collectors;
import java.util.zip.InflaterInputStream;

import javax.servlet.ServletException;
import javax.servlet.http.HttpServlet;
import javax.servlet.http.HttpServletRequest;
import javax.servlet.http.HttpServletResponse;

import org.eclipse.codewind.filewatchers.test.json.ChangedFileEntryJson;
import org.eclipse.codewind.filewatchers.test.json.FileChangeMsgJson;
import org.eclipse.codewind.filewatchers.test.json.ProjectToWatchJson;
import org.eclipse.codewind.filewatchers.test.json.ProjectToWatchListJson;
import org.eclipse.codewind.filewatchers.test.json.ProjectWatchStateStatusJson;

import com.fasterxml.jackson.annotation.JsonInclude.Include;
import com.fasterxml.jackson.databind.ObjectMapper;

@SuppressWarnings("serial")
public class CodewindTestApiServlet extends HttpServlet {

	private static final CodewindTestLogger log = CodewindTestLogger.getInstance();

	@Override
	protected void doGet(HttpServletRequest request, HttpServletResponse response)
			throws ServletException, IOException {

		if (CodewindTestState.getInstance().getChaosEngineering().failOrDelayResponse(request, response)) {
			return;
		}

		String requestURI = request.getRequestURI();

		List<ProjectToWatchJson> projectList = CodewindTestState.getInstance().getWatcherState().getProjects();

		ProjectToWatchListJson result = new ProjectToWatchListJson();
		result.setProjects(projectList);

		log.out("Received GET at " + requestURI + ", projectList size: " + projectList.size());

		ObjectMapper om = new ObjectMapper();
		om.setSerializationInclusion(Include.NON_NULL);

		response.setContentType("application/json");
		response.setStatus(HttpServletResponse.SC_OK);
		response.getWriter().println(om.writeValueAsString(result));
	}

	@Override
	protected void doPost(HttpServletRequest request, HttpServletResponse response)
			throws ServletException, IOException {

		if (CodewindTestState.getInstance().getChaosEngineering().failOrDelayResponse(request, response)) {
			return;
		}

		String requestURI = request.getRequestURI();

		log.out("Received POST at " + requestURI);

		List<String> urlParts = Arrays.asList(requestURI.split("/"));
		int projectsIndex = urlParts.indexOf("projects");
		if (projectsIndex == -1) {
			return;
		}

		String projectId = urlParts.get(projectsIndex + 1);

		long timestamp = Long.parseLong(request.getParameter("timestamp"));

		String bodyContents = CodewindTestUtils.readStringFromStream(request.getInputStream());

		ObjectMapper om = new ObjectMapper();
		FileChangeMsgJson fcmj = om.readValue(bodyContents, FileChangeMsgJson.class);

		byte[] barr = Base64.getDecoder().decode(fcmj.getMsg());

		InflaterInputStream dis = new InflaterInputStream(new ByteArrayInputStream(barr));

		String str = CodewindTestUtils.readStringFromStream(dis);
		dis.close();

		List<ChangedFileEntry> cfeList = Arrays.asList(om.readValue(str, ChangedFileEntryJson[].class)).stream()
				.map(e -> new ChangedFileEntry(e)).collect(Collectors.toList());

		CodewindTestState.getInstance().getChangeListFromWatcher()
				.addChangedFileEntries(new PostRequestContent(projectId, timestamp, cfeList));

		response.setContentType("application/json");
		response.setStatus(HttpServletResponse.SC_OK);

	}

	@Override
	protected void doPut(HttpServletRequest request, HttpServletResponse response)
			throws ServletException, IOException {

		if (CodewindTestState.getInstance().getChaosEngineering().failOrDelayResponse(request, response)) {
			return;
		}

		String requestURI = request.getRequestURI();

		String clientUuid = request.getParameter("clientUuid");
		if (clientUuid == null || clientUuid.trim().isEmpty()) {
			log.err("Put request missing clientUuid.");
			return;
		}

		log.out("Received PUT at " + requestURI);

		List<String> urlParts = Arrays.asList(requestURI.split("/"));
		int projectsIndex = urlParts.indexOf("projects");
		if (projectsIndex == -1) {
			log.err("Invalid PUT URL.");
			return;
		}

		String projectId = urlParts.get(projectsIndex + 1);

		int fileChangesIndex = urlParts.indexOf("file-changes");
		if (fileChangesIndex == -1) {
			log.err("Invalid PUT URL.");
			return;
		}

		String projectWatchStateId = urlParts.get(fileChangesIndex + 1);

		String body = CodewindTestUtils.readStringFromStream(request.getInputStream());

		ObjectMapper om = new ObjectMapper();
		ProjectWatchStateStatusJson status = om.readValue(body, ProjectWatchStateStatusJson.class);

		CodewindTestState.getInstance().getWatcherState().addWatchStatus(projectId, projectWatchStateId,
				status.isSuccess());
	}
}
