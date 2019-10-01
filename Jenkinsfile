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
                                #TAG_MAJOR = $TEST_BRANCH.tokenize(".")[0]​	
                                TAG_MAJOR = $TEST_BRANCH
                                echo "TAG_MAJOR is $TAG_MAJOR"	

                                #TAG_MINOR = $TEST_BRANCH.tokenize(".")[1]​	
                                #TAG_CUMULATIVE= $TAG_MAJOR.$TAG_MINOR	
                                    
                                
                            fi 
                        fi
                    '''

                }
            }
        } 
    }    
}
