pipeline {
    agent {
       label 'node4'
    }

    options {
          timeout(time: 10, unit: 'MINUTES')
    }

    stages {
        stage('Checkout') {
            steps {
               checkout scm
            }
        }

	    stage('Build Images') {
		    steps {
			    sh "make images"
		    }
	    }
    }
}
