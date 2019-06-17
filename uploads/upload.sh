#!/usr/bin/env bash

#*******************************************************************************
# Copyright (c) 2018, 2019 IBM Corporation and others.
# All rights reserved. This program and the accompanying materials
# are made available under the terms of the Eclipse Public License v2.0
# which accompanies this distribution, and is available at
# http://www.eclipse.org/legal/epl-v20.html
#
# Contributors:
#     IBM Corporation - initial API and implementation
#*******************************************************************************

# To be run from the repository root directory
# $artifact_name must be set and the file it points to must be in the working directory

DOWNLOADSITE_PATH="codewind-filewatchers/Filewatcherd-TypeScript/snapshots"
FILE_NAME="filewatcherd-node_1.0.0.tar.gz"


echo "Uploading \"$FILE_NAME\""

ssh genie.codewind@projects-storage.eclipse.org rm -rf /home/data/httpd/download.eclipse.org/codewind/$DOWNLOADSITE_PATH
ssh genie.codewind@projects-storage.eclipse.org mkdir -p /home/data/httpd/download.eclipse.org/codewind/$DOWNLOADSITE_PATH
scp -r ${WORKSPACE}/uploads/$FILE_NAME genie.codewind@projects-storage.eclipse.org:/home/data/httpd/download.eclipse.org/codewind/$DOWNLOADSITE_PATH
