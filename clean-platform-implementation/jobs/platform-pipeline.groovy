// Jenkins Job DSL for platform team CI/CD pipeline
// This creates all necessary jobs for the clean-platform implementation

// Common job configuration
def commonConfig = { job ->
    job.with {
        logRotator {
            numToKeep(30)
            artifactNumToKeep(10)
        }
        
        parameters {
            stringParam('BRANCH', 'main', 'Git branch to build')
            choiceParam('ENVIRONMENT', ['dev', 'staging', 'production'], 'Target environment')
            booleanParam('SKIP_TESTS', false, 'Skip running tests')
            booleanParam('FORCE_DEPLOY', false, 'Force deployment even if tests fail')
        }
        
        properties {
            buildDiscarder {
                strategy {
                    logRotator {
                        daysToKeepStr('30')
                        numToKeepStr('50')
                        artifactDaysToKeepStr('10')
                        artifactNumToKeepStr('10')
                    }
                }
            }
            
            githubProjectUrl('https://source.datanerd.us/platform-team/clean-platform-implementation')
        }
        
        wrappers {
            timestamps()
            colorizeOutput()
            timeout {
                absolute(60)
            }
            credentialsBinding {
                string('GHE_TOKEN', 'ghe-api-token')
                string('VAULT_TOKEN', 'vault-token')
                string('GRAND_CENTRAL_TOKEN', 'gc-api-token')
            }
        }
    }
}

// Main build pipeline
pipelineJob('platform-team/clean-platform-build') {
    description('Build and test clean-platform services')
    
    commonConfig(delegate)
    
    definition {
        cps {
            script('''
pipeline {
    agent any
    
    options {
        buildDiscarder(logRotator(numToKeepStr: '30'))
        timeout(time: 60, unit: 'MINUTES')
        timestamps()
        ansiColor('xterm')
    }
    
    environment {
        DOCKER_REGISTRY = 'cf-registry.nr-ops.net'
        IMAGE_PREFIX = 'platform-team'
        VAULT_ADDR = 'https://vault-prd1a.r10.us.nr-ops.net:8200'
    }
    
    stages {
        stage('Checkout') {
            steps {
                checkout([
                    $class: 'GitSCM',
                    branches: [[name: "${params.BRANCH}"]],
                    userRemoteConfigs: [[
                        url: 'git@source.datanerd.us:platform-team/clean-platform-implementation.git',
                        credentialsId: 'ghe-ssh-key'
                    ]]
                ])
            }
        }
        
        stage('Validate') {
            parallel {
                stage('Lint') {
                    steps {
                        sh \'\'\'
                            echo "Running linters..."
                            # Python linting
                            pip install flake8 black mypy
                            flake8 services/
                            black --check services/
                            mypy services/
                            
                            # Dockerfile linting
                            docker run --rm -i hadolint/hadolint < services/data-collector/Dockerfile
                            
                            # YAML linting
                            docker run --rm -v "${PWD}":/workdir mikefarah/yq eval '.' grandcentral.yml
                        \'\'\'
                    }
                }
                
                stage('Security Scan') {
                    steps {
                        sh \'\'\'
                            echo "Running security scans..."
                            # Dependency check
                            pip install safety
                            safety check -r services/data-collector/requirements.txt
                            
                            # Secret scanning
                            docker run --rm -v "${PWD}":/path zricethezav/gitleaks:latest detect --source="/path" --verbose
                        \'\'\'
                    }
                }
                
                stage('Schema Validation') {
                    steps {
                        sh \'\'\'
                            echo "Validating configurations..."
                            # Validate Grand Central config
                            curl -X POST https://grand-central.nr-ops.net/api/v1/validate \\
                                -H "X-Grand-Central-Auth: ${GRAND_CENTRAL_TOKEN}" \\
                                -H "Content-Type: application/json" \\
                                -d @grandcentral.yml
                        \'\'\'
                    }
                }
            }
        }
        
        stage('Test') {
            when {
                expression { params.SKIP_TESTS != true }
            }
            steps {
                sh \'\'\'
                    echo "Running tests..."
                    # Unit tests
                    cd services/data-collector
                    pip install -r requirements.txt pytest pytest-cov
                    pytest --cov=. --cov-report=xml --cov-report=html
                    
                    # Integration tests
                    cd ../../tests/integration
                    pytest test_api_integration.py
                \'\'\'
            }
            post {
                always {
                    junit '**/test-results/*.xml'
                    publishHTML([
                        allowMissing: false,
                        alwaysLinkToLastBuild: true,
                        keepAll: true,
                        reportDir: 'services/data-collector/htmlcov',
                        reportFiles: 'index.html',
                        reportName: 'Coverage Report'
                    ])
                }
            }
        }
        
        stage('Build Images') {
            steps {
                script {
                    def services = ['data-collector', 'data-processor', 'api-gateway']
                    def version = "${env.BUILD_NUMBER}-${env.GIT_COMMIT.take(7)}"
                    
                    services.each { service ->
                        sh """
                            cd services/${service}
                            docker build -t ${DOCKER_REGISTRY}/${IMAGE_PREFIX}/${service}:${version} .
                            docker tag ${DOCKER_REGISTRY}/${IMAGE_PREFIX}/${service}:${version} \\
                                ${DOCKER_REGISTRY}/${IMAGE_PREFIX}/${service}:latest-${params.BRANCH}
                        """
                    }
                }
            }
        }
        
        stage('Scan Images') {
            steps {
                script {
                    def services = ['data-collector', 'data-processor', 'api-gateway']
                    def version = "${env.BUILD_NUMBER}-${env.GIT_COMMIT.take(7)}"
                    
                    services.each { service ->
                        sh """
                            # Run Trivy scan
                            docker run --rm -v /var/run/docker.sock:/var/run/docker.sock \\
                                aquasec/trivy:latest image \\
                                --severity HIGH,CRITICAL \\
                                ${DOCKER_REGISTRY}/${IMAGE_PREFIX}/${service}:${version}
                            
                            # Run Dockle scan for CIS benchmarks
                            docker run --rm -v /var/run/docker.sock:/var/run/docker.sock \\
                                goodwithtech/dockle:latest \\
                                ${DOCKER_REGISTRY}/${IMAGE_PREFIX}/${service}:${version}
                        """
                    }
                }
            }
        }
        
        stage('Push Images') {
            when {
                anyOf {
                    branch 'main'
                    branch 'develop'
                    expression { params.FORCE_DEPLOY == true }
                }
            }
            steps {
                script {
                    docker.withRegistry("https://${DOCKER_REGISTRY}", 'cf-registry-creds') {
                        def services = ['data-collector', 'data-processor', 'api-gateway']
                        def version = "${env.BUILD_NUMBER}-${env.GIT_COMMIT.take(7)}"
                        
                        services.each { service ->
                            sh """
                                docker push ${DOCKER_REGISTRY}/${IMAGE_PREFIX}/${service}:${version}
                                docker push ${DOCKER_REGISTRY}/${IMAGE_PREFIX}/${service}:latest-${params.BRANCH}
                            """
                        }
                    }
                }
            }
        }
        
        stage('Deploy') {
            when {
                anyOf {
                    branch 'main'
                    expression { params.FORCE_DEPLOY == true }
                }
            }
            steps {
                script {
                    def version = "${env.BUILD_NUMBER}-${env.GIT_COMMIT.take(7)}"
                    
                    // Trigger Grand Central deployment
                    sh """
                        curl -X POST https://grand-central.nr-ops.net/api/v1/deploy \\
                            -H "X-Grand-Central-Auth: ${GRAND_CENTRAL_TOKEN}" \\
                            -H "Content-Type: application/json" \\
                            -d '{
                                "projectOrg": "platform-team",
                                "projectRepo": "clean-platform-implementation",
                                "environmentName": "${params.ENVIRONMENT}",
                                "version": "${version}",
                                "deployType": "deploy"
                            }'
                    """
                }
            }
        }
    }
    
    post {
        success {
            slackSend(
                color: 'good',
                message: "Build Successful: ${env.JOB_NAME} #${env.BUILD_NUMBER} (<${env.BUILD_URL}|Open>)"
            )
        }
        failure {
            slackSend(
                color: 'danger',
                message: "Build Failed: ${env.JOB_NAME} #${env.BUILD_NUMBER} (<${env.BUILD_URL}|Open>)"
            )
        }
        always {
            cleanWs()
        }
    }
}
            ''')
            sandbox(true)
        }
    }
    
    triggers {
        githubPush()
        cron('H 2 * * *')  // Daily build at 2 AM
    }
}

// Database migration job
job('platform-team/database-migration') {
    description('Run database migrations for clean-platform')
    
    commonConfig(delegate)
    
    parameters {
        choiceParam('MIGRATION_TYPE', ['up', 'down', 'status'], 'Migration direction')
        stringParam('TARGET_VERSION', '', 'Target migration version (empty for latest)')
    }
    
    scm {
        git {
            remote {
                url('git@source.datanerd.us:platform-team/clean-platform-implementation.git')
                credentials('ghe-ssh-key')
            }
            branch('${BRANCH}')
        }
    }
    
    steps {
        shell('''
            #!/bin/bash
            set -euo pipefail
            
            # Get database credentials from Vault
            DB_URL=$(vault kv get -field=endpoint terraform/platform-team/${ENVIRONMENT}/*/clean-platform/clean-platform-db-endpoint)
            DB_USER=$(vault kv get -field=master_username terraform/platform-team/${ENVIRONMENT}/*/clean-platform/clean-platform-db-endpoint)
            DB_PASS=$(vault kv get -field=master_password terraform/platform-team/${ENVIRONMENT}/*/clean-platform/clean-platform-db-endpoint)
            
            # Run migrations
            cd database/migrations
            
            case "${MIGRATION_TYPE}" in
                up)
                    echo "Running migrations up to ${TARGET_VERSION:-latest}..."
                    migrate -path . -database "postgresql://${DB_USER}:${DB_PASS}@${DB_URL}" up ${TARGET_VERSION}
                    ;;
                down)
                    echo "Rolling back to ${TARGET_VERSION}..."
                    migrate -path . -database "postgresql://${DB_USER}:${DB_PASS}@${DB_URL}" down ${TARGET_VERSION}
                    ;;
                status)
                    echo "Current migration status:"
                    migrate -path . -database "postgresql://${DB_USER}:${DB_PASS}@${DB_URL}" version
                    ;;
            esac
        ''')
    }
}

// Performance test job
job('platform-team/performance-tests') {
    description('Run performance tests against clean-platform')
    
    commonConfig(delegate)
    
    parameters {
        stringParam('TARGET_URL', '', 'Target environment URL')
        stringParam('USERS', '10', 'Number of concurrent users')
        stringParam('DURATION', '5m', 'Test duration')
        stringParam('RATE', '10', 'Requests per second')
    }
    
    scm {
        git {
            remote {
                url('git@source.datanerd.us:platform-team/clean-platform-implementation.git')
                credentials('ghe-ssh-key')
            }
            branch('${BRANCH}')
        }
    }
    
    steps {
        shell('''
            #!/bin/bash
            set -euo pipefail
            
            cd tests/performance
            
            # Run Locust performance tests
            locust --headless \\
                --users ${USERS} \\
                --spawn-rate ${RATE} \\
                --time ${DURATION} \\
                --host ${TARGET_URL} \\
                --html performance-report.html \\
                --csv performance-results
            
            # Check if performance thresholds are met
            python3 check_performance_thresholds.py performance-results_stats.csv
        ''')
    }
    
    publishers {
        archiveArtifacts {
            pattern('tests/performance/performance-*')
            allowEmpty(false)
        }
        
        publishHTML {
            reportDir('tests/performance')
            reportFiles('performance-report.html')
            reportName('Performance Test Report')
            keepAll(true)
        }
    }
}

// Create a view for all platform jobs
listView('Platform Team') {
    description('All jobs for the platform team')
    filterBuildQueue()
    filterExecutors()
    
    jobs {
        regex('platform-team/.*')
    }
    
    columns {
        status()
        weather()
        name()
        lastSuccess()
        lastFailure()
        lastDuration()
        buildButton()
    }
}