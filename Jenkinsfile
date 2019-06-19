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
                    '''
                    
                    dir('org.eclipse.codewind.filewatchers.core') { sh 'mvn clean install' }
                    dir('org.eclipse.codewind.filewatchers.standalonenio') { sh 'mvn clean install' }
                    dir('org.eclipse.codewind.filewatchers.eclipse') { sh 'mvn clean package' }                    
                }
            }
        } 
        
        stage('Deploy') {
            steps {
                sshagent ( ['projects-storage.eclipse.org-bot-ssh']) {
                  println("Deploying codewind-filewatchers to downoad area...")
 
                 sh '''                 	
                  	ssh genie.codewind@projects-storage.eclipse.org rm -rf /home/data/httpd/download.eclipse.org/codewind/codewind-filewatchers/snapshots
                  	ssh genie.codewind@projects-storage.eclipse.org mkdir -p /home/data/httpd/download.eclipse.org/codewind/codewind-filewatchers/snapshots
                  	scp -r ${WORKSPACE}/org.eclipse.codewind.filewatchers.core/target/org.eclipse.codewind.filewatchers*.jar genie.codewind@projects-storage.eclipse.org:/home/data/httpd/download.eclipse.org/codewind/codewind-filewatchers/snapshots
                  	scp -r ${WORKSPACE}/org.eclipse.codewind.filewatchers.standalonenio/target/org.eclipse.codewind.filewatchers*.jar genie.codewind@projects-storage.eclipse.org:/home/data/httpd/download.eclipse.org/codewind/codewind-filewatchers/snapshots
                  	scp -r ${WORKSPACE}/org.eclipse.codewind.filewatchers.eclipse/target/org.eclipse.codewind.filewatchers*.jar genie.codewind@projects-storage.eclipse.org:/home/data/httpd/download.eclipse.org/codewind/codewind-filewatchers/snapshots
                  '''
                }
            }
        }       
    }    
}
