#!groovy​

// Preamble -------------------------------------------------------------------

IS_MASTER_BRANCH = env.BRANCH_NAME == "master"
IS_RELEASE_BRANCH = (env.BRANCH_NAME ==~ /\d+\.\d+\.\d+/)


echo "Branch is ${env.BRANCH_NAME}"
echo "Is master branch build ? ${IS_MASTER_BRANCH}"
echo "Is release branch build ? ${IS_RELEASE_BRANCH}"

// https://stackoverflow.com/a/44902622
def CRON_STRING = ""
// https://jenkins.io/doc/book/pipeline/syntax/#cron-syntax
if (IS_MASTER_BRANCH) {
    // Build daily between 0300-0559
    CRON_STRING = "H H(3-5) * * *"
} else {
    // Delete me ASAP
    CRON_STRING = "H */2 * * *"
}


// Pipeline -------------------------------------------------------------------

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
    resources:
      limits:
        memory: "2Gi"
        cpu: "1"
      requests:
        memory: "2Gi"
        cpu: "1"      
  - name: go
    image: golang:1.12-stretch
    tty: true
    command:
    - cat
    resources:
      limits:
        memory: "2Gi"
        cpu: "1"
      requests:
        memory: "2Gi"
        cpu: "1"    
"""
        }
    }

    options {
        timestamps()
        skipStagesAfterUnstable()
    }


    triggers {
        cron(CRON_STRING)
    }


    environment {
        // https://stackoverflow.com/a/43264045
        HOME="."
    }

    stages {

        stage('Build TypeScript Filewatcher') {

            options {
                timeout(time: 30, unit: 'MINUTES') 
            }

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

        stage('Run tests') {

            options {
                timeout(time: 120, unit: 'MINUTES') 
            }

            steps {
                container("go") {
                    dir ("Tests") {
                        sh '''#!/usr/bin/env bash

                            echo "TODO: Remove me: print env"
                            printenv

                            set -euo pipefail

                            echo
                            echo "Download Java and add to path"
                            echo
                            export STEP_ROOT_PATH=`pwd`
                            curl -LO https://github.com/AdoptOpenJDK/openjdk8-binaries/releases/download/jdk8u242-b08_openj9-0.18.1/OpenJDK8U-jdk_x64_linux_openj9_8u242b08_openj9-0.18.1.tar.gz
                            tar xzf OpenJDK8U-jdk_x64_linux_openj9_8u242b08_openj9-0.18.1.tar.gz
                            cd jdk8u242-b08
                            export JAVA_HOME=`pwd`
                            cd bin/
                            export PATH=`pwd`:$PATH

                            echo 
                            echo "Download Maven and add to path"
                            echo
                            cd $STEP_ROOT_PATH
                            curl -LO http://mirror.dsrg.utoronto.ca/apache/maven/maven-3/3.6.3/binaries/apache-maven-3.6.3-bin.tar.gz
                            tar xzf apache-maven-3.6.3-bin.tar.gz
                            cd apache-maven-3.6.3/bin
                            export PATH=`pwd`:$PATH

                            echo 
                            echo "Run Go tests"
                            echo
                            cd $STEP_ROOT_PATH/
                            ./run_tests_go_filewatcher.sh

                        '''
                    }
                }
                container("filewatcherd-ts-builder") {
                    dir ("Tests") {
                        sh '''#!/usr/bin/env bash

                            set -euo pipefail

                            echo
                            echo "Download Java and add to path"
                            echo
                            export STEP_ROOT_PATH=`pwd`
                            curl -LO https://github.com/AdoptOpenJDK/openjdk8-binaries/releases/download/jdk8u242-b08_openj9-0.18.1/OpenJDK8U-jdk_x64_linux_openj9_8u242b08_openj9-0.18.1.tar.gz
                            tar xzf OpenJDK8U-jdk_x64_linux_openj9_8u242b08_openj9-0.18.1.tar.gz
                            cd jdk8u242-b08
                            export JAVA_HOME=`pwd`
                            cd bin/
                            export PATH=`pwd`:$PATH

                            echo 
                            echo "Download Maven and add to path"
                            echo
                            cd $STEP_ROOT_PATH
                            curl -LO http://mirror.dsrg.utoronto.ca/apache/maven/maven-3/3.6.3/binaries/apache-maven-3.6.3-bin.tar.gz
                            tar xzf apache-maven-3.6.3-bin.tar.gz
                            cd apache-maven-3.6.3/bin
                            export PATH=`pwd`:$PATH

                            echo 
                            echo "Run Node tests"
                            echo

                            cd $STEP_ROOT_PATH/
                            ./run_tests_node_filewatcher.sh

                        '''
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
                timeout(time: 120, unit: 'MINUTES') 
                retry(3) 
            }

            agent any

            steps {
                sshagent (['projects-storage.eclipse.org-bot-ssh']) {
                    println("Deploying codewind-filewatchers to download area...")
                    unstash "output"

                    sh '''
                        export SSH_HOST="genie.codewind@projects-storage.eclipse.org"
                        export TARGET_DIR_PARENT="/home/data/httpd/archive.eclipse.org/codewind/codewind-filewatcher-ts/${GIT_BRANCH}/"
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
