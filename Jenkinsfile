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
                    println("Starting codewind-filewatchers build ...")
                        
                    def sys_info = sh(script: "uname -a", returnStdout: true).trim()
                    println("System information: ${sys_info}")
                    println("JAVE_HOME: ${JAVA_HOME}")
                    
                    sh '''
                        java -version
                        which java    
                        
                        # Place hoder for build script
                    '''
                }
            }
        } 
        
        stage('Deploy') {
            steps {
                sshagent ( ['projects-storage.eclipse.org-bot-ssh']) {
                  println("Deploying codewind-filewatchers to downoad area...")
 
                 sh '''          
                    if [ -z $CHANGE_ID ]; then
    					UPLOAD_DIR="$GIT_BRANCH/$BUILD_ID"
					else
    					UPLOAD_DIR="pr/$CHANGE_ID/$BUILD_ID"
					fi
        	
                  	ssh genie.codewind@projects-storage.eclipse.org rm -rf /home/data/httpd/download.eclipse.org/codewind/codewind-filewatchers/$(UPLOAD_DIR)
                  	ssh genie.codewind@projects-storage.eclipse.org mkdir -p /home/data/httpd/download.eclipse.org/codewind/codewind-filewatchers/$(UPLOAD_DIR)
                  	
                  	# Place hoder for deploy script
                  '''
                }
            }
        }       
    }    
}
