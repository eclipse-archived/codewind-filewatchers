/*******************************************************************************
* Copyright (c) 2019, 2020 IBM Corporation and others.
* All rights reserved. This program and the accompanying materials
* are made available under the terms of the Eclipse Public License v2.0
* which accompanies this distribution, and is available at
* http://www.eclipse.org/legal/epl-v20.html
*
* Contributors:
*     IBM Corporation - initial API and implementation
*******************************************************************************/

import clientns from "./client";

/** This file will start the filewatcher when doing standalone development eg outside the VSCode integration scenario */

clientns(process.env.CODEWIND_URL_ROOT, undefined, undefined, process.env.MOCK_CWCTL_INSTALLER_PATH);
