#!groovy​

pipeline {
    agent {
        kubernetes {
            label 'filewatcherd-buildpod'
            yaml """
apiVersion: v1
kind: Pod
spec:
  containers:
  - name: filewatcherd-ts-builder
    image: node:lts
    tty: true
    command:
      - cat
"""
        }
    }

    options {
        timestamps()
        skipStagesAfterUnstable()
    }

    environment {
        // https://stackoverflow.com/a/43264045
        HOME="."
    }

    stages {

        stage('Build TypeScript Filewatcher') {
            steps {
                container("filewatcherd-ts-builder") {
                    dir ("Filewatcherd-TypeScript") {
                        sh '''#!/usr/bin/env bash
                            ./npm-package.sh
                        '''
                        stash includes: "filewatcherd*.tar.gz", name: "output"
                    }
                }
            }
        }

        stage('Deploy') {
            // This when clause disables PR build uploads; you may comment this out if you want your build uploaded.
            when {
                beforeAgent true
                not {
                    changeRequest()
                }
            }

            options {
                skipDefaultCheckout()
                timeout(time: 30, unit: 'MINUTES') 
            }

            agent any

            steps {
                sshagent (['projects-storage.eclipse.org-bot-ssh']) {
                    println("Deploying codewind-filewatchers to download area...")
                    unstash "output"

                    retry(3) {
                        sh '''
                            export SSH_HOST="genie.codewind@projects-storage.eclipse.org"
                            export TARGET_DIR_PARENT="/home/data/httpd/download.eclipse.org/codewind/codewind-filewatcher-ts/${GIT_BRANCH}/"
                            export TARGET_DIR="${TARGET_DIR_PARENT}/${BUILD_ID}"
                            export LATEST_DIR="${TARGET_DIR_PARENT}/latest"
                            export ARTIFACT_NAME="$(basename "filewatcherd*.tar.gz")"
                            export LATEST_ARTIFACT_NAME="filewatcherd-node_latest.tar.gz"
                            export BUILD_INFO="build_info.properties"

                            echo "# Build date: $(date +%F-%T)" >> $BUILD_INFO
                            echo "build_info.url=$BUILD_URL" >> $BUILD_INFO
                            SHA1=$(sha1sum ${ARTIFACT_NAME} | cut -d ' ' -f 1)
                            echo "build_info.SHA-1=${SHA1}" >> $BUILD_INFO

                            set -x
                            ssh $SSH_HOST mkdir -p $TARGET_DIR
                            scp $ARTIFACT_NAME ${SSH_HOST}:${TARGET_DIR}

                            cp -v $ARTIFACT_NAME $LATEST_ARTIFACT_NAME
                            ssh $SSH_HOST mkdir -p $LATEST_DIR
                            scp $LATEST_ARTIFACT_NAME ${SSH_HOST}:${LATEST_DIR}
                            scp $BUILD_INFO ${SSH_HOST}:${LATEST_DIR}
                        '''
                    }
                }
            }
        }
    }
}
