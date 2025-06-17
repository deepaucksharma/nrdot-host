#!/usr/bin/env python3
"""
Compliance Checker for Clean Platform Implementation
Validates implementation against all platform guidelines from .md files
"""

import os
import sys
import yaml
import json
import re
from pathlib import Path
from typing import Dict, List, Set, Tuple
import ast

class ComplianceChecker:
    def __init__(self, docs_path: str = "/home/deepak/src/nr-platform-docs-gh-pages"):
        self.docs_path = Path(docs_path)
        self.violations = []
        self.warnings = []
        self.errors = []
        self.guidelines = self._load_guidelines()
    
    def _load_guidelines(self) -> Dict[str, List[str]]:
        """Load requirements from platform documentation"""
        guidelines = {
            'grand_central': [],
            'container_security': [],
            'secrets_management': [],
            'monitoring': [],
            'database': [],
            'change_management': [],
            'jenkins': [],
            'github': []
        }
        
        # Extract requirements from CLAUDE.md
        claude_md = self.docs_path / "CLAUDE.md"
        if claude_md.exists():
            with open(claude_md, 'r') as f:
                content = f.read()
                # Extract key requirements
                if "Grand Central" in content:
                    guidelines['grand_central'].extend([
                        'Must use Grand Central for all deployments',
                        'Requires X-Grand-Central-Auth header',
                        'Must register project before deployment',
                        'deployment_hooks required for production'
                    ])
        
        return guidelines
    
    def check_grand_central_compliance(self) -> None:
        """Validate Grand Central configuration"""
        gc_files = ['grandcentral.yml', 'grandcentral-enhanced.yml']
        
        for gc_file in gc_files:
            if not Path(gc_file).exists():
                continue
                
            with open(gc_file, 'r') as f:
                try:
                    config = yaml.safe_load(f)
                except yaml.YAMLError as e:
                    self.errors.append(f"{gc_file}: Invalid YAML - {e}")
                    continue
            
            # Required top-level fields
            required_fields = [
                'project_name', 'team_name', 'deploy_mechanism',
                'environments', 'docker_build'
            ]
            
            for field in required_fields:
                if field not in config:
                    self.violations.append(f"{gc_file}: Missing required field '{field}'")
            
            # Check deployment hooks
            if 'base_environment' in config:
                hooks = config.get('base_environment', {}).get('deployment_hooks', {})
                if 'pre_deploy' not in hooks:
                    self.violations.append(f"{gc_file}: Missing pre_deploy hooks")
                if 'post_deploy' not in hooks:
                    self.violations.append(f"{gc_file}: Missing post_deploy hooks")
                
                # Validate hook scripts exist
                for hook_type in ['pre_deploy', 'post_deploy']:
                    if hook_type in hooks:
                        for hook in hooks[hook_type]:
                            if 'script' in hook:
                                script_path = hook['script']
                                if not Path(script_path).exists():
                                    self.errors.append(f"{gc_file}: Hook script missing: {script_path}")
            
            # Check environments
            for env in config.get('environments', []):
                if 'name' not in env:
                    self.violations.append(f"{gc_file}: Environment missing 'name'")
                
                # Production specific checks
                if env.get('name') == 'production':
                    if not env.get('deployment_windows'):
                        self.warnings.append(f"{gc_file}: Production missing deployment_windows")
                    if not env.get('change_management', {}).get('enabled'):
                        self.violations.append(f"{gc_file}: Production must have change_management enabled")
    
    def check_dockerfile_compliance(self) -> None:
        """Check all Dockerfiles for security compliance"""
        for dockerfile_path in Path('.').rglob('Dockerfile'):
            with open(dockerfile_path, 'r') as f:
                content = f.read()
                lines = content.split('\n')
            
            # Check base image compliance
            from_lines = [l for l in lines if l.strip().startswith('FROM')]
            for from_line in from_lines:
                if 'cf-registry.nr-ops.net/newrelic' not in from_line:
                    if 'cf-registry.nr-ops.net/platform-team' not in from_line:
                        self.violations.append(
                            f"{dockerfile_path}: Must use FIPS-compliant base image from cf-registry.nr-ops.net"
                        )
                
                # Check for :latest tag
                if ':latest' in from_line:
                    self.warnings.append(f"{dockerfile_path}: Avoid using :latest tag")
            
            # Check for non-root user
            user_lines = [l for l in lines if l.strip().startswith('USER')]
            if not user_lines:
                self.violations.append(f"{dockerfile_path}: Must run as non-root user")
            else:
                for user_line in user_lines:
                    user_match = re.search(r'USER\s+(\d+)', user_line)
                    if user_match:
                        uid = int(user_match.group(1))
                        if uid < 10000:
                            self.violations.append(
                                f"{dockerfile_path}: User ID must be >= 10000, found {uid}"
                            )
            
            # Check for security best practices
            if 'apt-get install' in content and 'apt-get clean' not in content:
                self.warnings.append(f"{dockerfile_path}: Should clean apt cache after install")
            
            if 'COPY . .' in content or 'ADD . .' in content:
                self.warnings.append(f"{dockerfile_path}: Avoid copying entire directory")
    
    def check_kubernetes_compliance(self) -> None:
        """Check Kubernetes manifests for platform compliance"""
        for yaml_path in Path('.').rglob('*.yaml'):
            if 'node_modules' in str(yaml_path):
                continue
                
            with open(yaml_path, 'r') as f:
                try:
                    # Handle multi-document YAML
                    for doc in yaml.safe_load_all(f):
                        if not doc:
                            continue
                        self._validate_k8s_resource(doc, yaml_path)
                except yaml.YAMLError as e:
                    self.errors.append(f"{yaml_path}: Invalid YAML - {e}")
    
    def _validate_k8s_resource(self, resource: Dict, file_path: Path) -> None:
        """Validate individual Kubernetes resource"""
        kind = resource.get('kind', '')
        
        if kind == 'Deployment':
            spec = resource.get('spec', {}).get('template', {}).get('spec', {})
            
            # Check security context
            if 'securityContext' not in spec:
                self.violations.append(f"{file_path}: Deployment missing pod securityContext")
            else:
                sec_ctx = spec['securityContext']
                if not sec_ctx.get('runAsNonRoot'):
                    self.violations.append(f"{file_path}: Must set runAsNonRoot: true")
                if sec_ctx.get('runAsUser', 0) < 10000:
                    self.violations.append(f"{file_path}: runAsUser must be >= 10000")
            
            # Check containers
            for container in spec.get('containers', []):
                # Container security context
                container_sec = container.get('securityContext', {})
                if not container_sec.get('allowPrivilegeEscalation') == False:
                    self.violations.append(
                        f"{file_path}: Container {container.get('name')} must set allowPrivilegeEscalation: false"
                    )
                
                # Resource limits
                if 'resources' not in container:
                    self.violations.append(
                        f"{file_path}: Container {container.get('name')} missing resource limits"
                    )
                
                # Image compliance
                image = container.get('image', '')
                if not image.startswith('cf-registry.nr-ops.net/'):
                    self.violations.append(
                        f"{file_path}: Container {container.get('name')} must use cf-registry.nr-ops.net"
                    )
                
                # Health checks
                if 'livenessProbe' not in container:
                    self.warnings.append(
                        f"{file_path}: Container {container.get('name')} missing livenessProbe"
                    )
                if 'readinessProbe' not in container:
                    self.warnings.append(
                        f"{file_path}: Container {container.get('name')} missing readinessProbe"
                    )
        
        elif kind == 'NetworkPolicy':
            # Check for egress rules
            spec = resource.get('spec', {})
            if 'egress' in spec:
                for egress in spec['egress']:
                    if 'to' in egress:
                        for to_rule in egress['to']:
                            # Check for overly permissive rules
                            if 'ipBlock' in to_rule:
                                cidr = to_rule['ipBlock'].get('cidr', '')
                                if cidr == '0.0.0.0/0':
                                    self.violations.append(
                                        f"{file_path}: NetworkPolicy has overly permissive egress (0.0.0.0/0)"
                                    )
    
    def check_service_code_compliance(self) -> None:
        """Check service code for required integrations"""
        service_dirs = ['services/data-collector', 'services/data-processor', 'services/api-gateway']
        
        for service_dir in service_dirs:
            if not Path(service_dir).exists():
                continue
            
            # Check for required files
            required_files = ['app.py', 'requirements.txt', 'Dockerfile']
            for req_file in required_files:
                if not (Path(service_dir) / req_file).exists():
                    self.violations.append(f"{service_dir}: Missing required file {req_file}")
            
            # Check Python code
            app_py = Path(service_dir) / 'app.py'
            if app_py.exists():
                with open(app_py, 'r') as f:
                    content = f.read()
                
                # Check for required integrations
                if 'from prometheus_client import' not in content:
                    self.violations.append(f"{service_dir}: Missing Prometheus metrics integration")
                
                if '/healthz' not in content:
                    self.violations.append(f"{service_dir}: Missing /healthz endpoint")
                
                if '/readyz' not in content:
                    self.violations.append(f"{service_dir}: Missing /readyz endpoint")
                
                # Check for error handling
                if 'try:' not in content:
                    self.warnings.append(f"{service_dir}: No error handling found")
                
                # Check for logging
                if 'import logging' not in content:
                    self.violations.append(f"{service_dir}: Missing logging configuration")
    
    def check_terraform_compliance(self) -> None:
        """Check Terraform configurations"""
        for tf_file in Path('.').rglob('*.tf'):
            with open(tf_file, 'r') as f:
                content = f.read()
            
            # Check for required tags
            if 'resource' in content and 'tags' not in content:
                self.warnings.append(f"{tf_file}: Resources should have tags")
            
            # Check for hardcoded values
            if re.search(r'ami-[a-f0-9]{17}', content):
                self.violations.append(f"{tf_file}: Hardcoded AMI ID found")
            
            # Check for proper backend configuration
            if 'terraform {' in content and 'backend' not in content:
                if 'main.tf' in str(tf_file):
                    self.warnings.append(f"{tf_file}: Missing backend configuration")
    
    def check_secret_management(self) -> None:
        """Check for exposed secrets and proper secret management"""
        exclude_dirs = {'.git', 'node_modules', '.terraform', '__pycache__', 'venv'}
        
        # Patterns that might indicate secrets
        secret_patterns = [
            (r'(?i)(api[_-]?key|apikey)\s*[:=]\s*["\']?[a-zA-Z0-9]{20,}', 'API key'),
            (r'(?i)(secret|password|passwd|pwd)\s*[:=]\s*["\']?[^\s]{8,}', 'Password'),
            (r'(?i)token\s*[:=]\s*["\']?[a-zA-Z0-9]{20,}', 'Token'),
            (r'-----BEGIN (RSA |EC )?PRIVATE KEY-----', 'Private key'),
            (r'aws_access_key_id\s*=\s*[A-Z0-9]{20}', 'AWS access key'),
            (r'aws_secret_access_key\s*=\s*[a-zA-Z0-9/+=]{40}', 'AWS secret key')
        ]
        
        for file_path in Path('.').rglob('*'):
            if file_path.is_dir() or any(exc in str(file_path) for exc in exclude_dirs):
                continue
            
            try:
                with open(file_path, 'r', encoding='utf-8', errors='ignore') as f:
                    content = f.read()
                
                for pattern, secret_type in secret_patterns:
                    if re.search(pattern, content):
                        # Check if it's a placeholder
                        if not any(placeholder in content for placeholder in 
                                 ['YOUR_', 'PLACEHOLDER', 'EXAMPLE', 'example', 'TODO']):
                            self.violations.append(
                                f"{file_path}: Potential {secret_type} exposed"
                            )
            except:
                pass  # Skip binary files
    
    def check_monitoring_compliance(self) -> None:
        """Check monitoring and observability compliance"""
        # Check for monitoring configuration
        monitoring_files = list(Path('.').rglob('*monitoring*'))
        if not monitoring_files:
            self.violations.append("No monitoring configuration found")
        
        # Check for required alert configurations
        alerts_tf = Path('terraform/monitoring/alerts.tf')
        if alerts_tf.exists():
            with open(alerts_tf, 'r') as f:
                content = f.read()
            
            required_alerts = ['service_availability', 'response_time', 'error_rate']
            for alert in required_alerts:
                if alert not in content:
                    self.violations.append(f"Missing required alert: {alert}")
    
    def check_required_integrations(self) -> None:
        """Check for required platform integrations"""
        required_integrations = {
            'services/common/change_management_client.py': 'Change Management integration',
            'services/common/grand_central_client.py': 'Grand Central client',
            'services/common/circuit_breaker.py': 'Circuit breaker pattern',
            'services/common/tracing.py': 'Distributed tracing'
        }
        
        for file_path, description in required_integrations.items():
            if not Path(file_path).exists():
                self.violations.append(f"Missing required integration: {description} ({file_path})")
            else:
                # Check if the file has actual implementation
                with open(file_path, 'r') as f:
                    content = f.read()
                    if len(content.strip()) < 100 or 'TODO' in content or 'pass' in content:
                        self.warnings.append(f"Incomplete implementation: {description}")
    
    def generate_report(self) -> bool:
        """Generate compliance report"""
        print("=" * 80)
        print("COMPLIANCE VALIDATION REPORT - Clean Platform Implementation")
        print("=" * 80)
        print(f"\nSummary:")
        print(f"  Violations (must fix): {len(self.violations)}")
        print(f"  Warnings (should fix): {len(self.warnings)}")
        print(f"  Errors (blocking): {len(self.errors)}")
        
        if self.errors:
            print("\n❌ ERRORS (Blocking Issues):")
            for error in sorted(set(self.errors)):
                print(f"  • {error}")
        
        if self.violations:
            print("\n❌ VIOLATIONS (Must Fix):")
            for violation in sorted(set(self.violations)):
                print(f"  • {violation}")
        
        if self.warnings:
            print("\n⚠️  WARNINGS (Should Fix):")
            for warning in sorted(set(self.warnings)):
                print(f"  • {warning}")
        
        # Compliance score
        total_issues = len(self.violations) + len(self.errors)
        if total_issues == 0:
            print("\n✅ COMPLIANCE STATUS: PASSED")
            compliance_score = 100
        else:
            compliance_score = max(0, 100 - (total_issues * 5))
            print(f"\n❌ COMPLIANCE STATUS: FAILED (Score: {compliance_score}/100)")
        
        # Recommendations
        print("\n📋 RECOMMENDATIONS:")
        if self.violations or self.errors:
            print("  1. Fix all violations and errors before deployment")
            print("  2. Address warnings to improve quality")
            print("  3. Run compliance check in CI/CD pipeline")
            print("  4. Review platform documentation for requirements")
        else:
            print("  1. Address warnings to achieve 100% compliance")
            print("  2. Set up automated compliance monitoring")
        
        print("\n" + "=" * 80)
        
        return total_issues == 0
    
    def run_all_checks(self) -> bool:
        """Run all compliance checks"""
        print("Running compliance checks...")
        
        self.check_grand_central_compliance()
        self.check_dockerfile_compliance()
        self.check_kubernetes_compliance()
        self.check_service_code_compliance()
        self.check_terraform_compliance()
        self.check_secret_management()
        self.check_monitoring_compliance()
        self.check_required_integrations()
        
        return self.generate_report()


def main():
    """Main entry point"""
    import argparse
    
    parser = argparse.ArgumentParser(description='Check platform compliance')
    parser.add_argument('--docs-path', default='/home/deepak/src/nr-platform-docs-gh-pages',
                      help='Path to platform documentation')
    parser.add_argument('--fix', action='store_true',
                      help='Attempt to auto-fix issues (not implemented)')
    
    args = parser.parse_args()
    
    checker = ComplianceChecker(docs_path=args.docs_path)
    success = checker.run_all_checks()
    
    # Exit with appropriate code
    sys.exit(0 if success else 1)


if __name__ == '__main__':
    main()