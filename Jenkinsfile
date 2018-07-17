pipeline {
  agent {
    node {
      label 'node'
    }

  }
  stages {
    stage('Run tests') {
      parallel {
        stage('k8s-1.10.3-dev') {
          environment {
            TARGET = 'k8s-1.10.3-dev'
          }
          steps {
            timeout(unit: 'MINUTES', time: 180) {
              timestamps() {
                sh '''#!/bin/bash
set -o pipefail

bash automation/test.sh 2>&1 | tee ${WORKSPACE}/${TARGET}-console.log'''
              }

            }

          }
        }
        stage('k8s-1.10.3-release') {
          environment {
            TARGET = 'k8s-1.10.3-release'
          }
          steps {
            timeout(time: 180, unit: 'MINUTES') {
              timestamps() {
                sh '''#!/bin/bash
set -o pipefail

bash automation/test.sh 2>&1 | tee ${WORKSPACE}/${TARGET}-console.log'''
              }

            }

          }
        }
      }
    }
    stage('Clean Workspace') {
      steps {
        cleanWs(cleanWhenAborted: true, cleanWhenFailure: true, cleanWhenNotBuilt: true, cleanWhenSuccess: true, cleanWhenUnstable: true)
      }
    }
  }
}