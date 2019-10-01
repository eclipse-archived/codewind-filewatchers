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
                        echo "Branch is $GIT_BRANCH"

                        if [[ $GIT_BRANCH == "master" ]] || [[ $GIT_BRANCH =~ ^([0-9]+\\.[0-9]+) ]]; then
                            echo "Branch is $GIT_BRANCH"

                            if [[ $GIT_BRANCH =~ ^([0-9]+\\.[0-9]+) ]]; then	
                                TAG_MAJOR = $GIT_BRANCH.tokenize(".")[0]​	

                                echo "TAG_MAJOR is $TAG_MAJOR"	

                                #TAG_MINOR = $GIT_BRANCH.tokenize(".")[1]​	
                                #TAG_CUMULATIVE= $TAG_MAJOR.$TAG_MINOR	
                                    
                                
                            fi 
                        fi
                    '''

                }
            }
        } 
    }    
}
