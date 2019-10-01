#!groovy​

pipeline {
    agent any
    
    tools {
        jdk 'oracle-jdk8-latest'
        maven 'apache-maven-latest'
    }
    
    options {
        timestamps() 
        skipStagesAfterUnstable()
    }
    
    stages {

        stage('Build') {
            steps {
                script {
                    println("Starting Test build ...")
                        
                    def sys_info = sh(script: "uname -a", returnStdout: true).trim()
                    println("System information: ${sys_info}")
                    println("JAVE_HOME: ${JAVA_HOME}")
                    
                    sh '''#!/usr/bin/env bash
                        export TEST_BRANCH="0.1.0" 
                        echo "Test Branch is $TEST_BRANCH"

                        if [[ $TEST_BRANCH == "master" ]] || [[ $TEST_BRANCH =~ ^([0-9]+\\.[0-9]+) ]]; then

                            declare -a DOCKER_IMAGE_ARRAY=("codewind-initialize-amd64" 
                                                        "codewind-performance-amd64" 
                                                        "codewind-pfe-amd64")

                            #chmod u+x ./script/publish.sh

                            for i in "${DOCKER_IMAGE_ARRAY[@]}"
                            do
                                echo "Publishing $REGISTRY/$i:$TAG"
                                #./script/publish.sh $i $REGISTRY $TAG
                                fi 
                            done

                            if [[ $TEST_BRANCH =~ ^([0-9]+\\.[0-9]+) ]]; then
                                IFS='.' # set '.' as delimiter
                                read -ra TOKENS <<< "$TEST_BRANCH"    
                                IFS=' ' # reset delimiter
                                export TAG_CUMULATIVE=${TOKENS[0]}.${TOKENS[1]}

                                for i in "${DOCKER_IMAGE_ARRAY[@]}"
                                do
                                    echo "Publishing $REGISTRY/$i:$TAG_CUMULATIVE"
                                #    ./script/publish.sh $i $REGISTRY $TAG_CUMULATIVE
                                done
                            fi
                        else
                            echo "Skip publishing docker images for $TEST_BRANCH branch"
                        fi
                    '''

                }
            }
        } 
    }    
}
