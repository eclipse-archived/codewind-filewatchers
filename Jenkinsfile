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
                            echo "Branch is $TEST_BRANCH"

                            if [[ $TEST_BRANCH =~ ^([0-9]+\\.[0-9]+) ]]; then	
                                IFS='.' # set '.' as delimiter
                                read -ra RELEASE <<< "$TEST_BRANCH"    
                                IFS=' ' # reset to default value after usage

                                #export TAG_CUMULATIVE=${RELEASE[0]}.${RELEASE[1]}
                                export TAG_CUMULATIVE=${RELEASE[@]:0:2}      
                                echo "TAG_CUMULATIVE is $TAG_CUMULATIVE"	
                            fi 
                        fi
                    '''

                }
            }
        } 
    }    
}
