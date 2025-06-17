#!/usr/bin/env python3
"""
Validate Grand Central configuration against platform requirements
"""

import sys
import yaml
import json
from pathlib import Path
from jsonschema import validate, ValidationError

# Grand Central configuration schema based on platform documentation
GRAND_CENTRAL_SCHEMA = {
    "type": "object",
    "required": [
        "project_name",
        "team_name",
        "deploy_mechanism",
        "environments"
    ],
    "properties": {
        "project_name": {
            "type": "string",
            "pattern": "^[a-z0-9-]+$",
            "minLength": 3,
            "maxLength": 50
        },
        "team_name": {
            "type": "string",
            "pattern": "^[a-z0-9-]+$"
        },
        "deploy_mechanism": {
            "type": "string",
            "enum": ["kubernetes", "terraform", "ecs"]
        },
        "slack_channel": {
            "type": "string",
            "pattern": "^#[a-z0-9-]+$"
        },
        "docker_build": {
            "type": "object",
            "properties": {
                "registry": {
                    "type": "string",
                    "pattern": "^cf-registry\\.nr-ops\\.net"
                },
                "build_args": {
                    "type": "object",
                    "properties": {
                        "BASE_IMAGE": {
                            "type": "string",
                            "pattern": "cf-registry\\.nr-ops\\.net/newrelic/.*-fips"
                        }
                    }
                }
            }
        },
        "environments": {
            "type": "array",
            "minItems": 1,
            "items": {
                "type": "object",
                "required": ["name"],
                "properties": {
                    "name": {
                        "type": "string",
                        "enum": ["development", "staging", "production"]
                    },
                    "kubernetes": {
                        "type": "object",
                        "properties": {
                            "namespace": {"type": "string"},
                            "resources": {
                                "type": "object",
                                "required": ["requests", "limits"],
                                "properties": {
                                    "requests": {
                                        "type": "object",
                                        "required": ["cpu", "memory"]
                                    },
                                    "limits": {
                                        "type": "object",
                                        "required": ["memory"]
                                    }
                                }
                            }
                        }
                    }
                }
            }
        },
        "change_management": {
            "type": "object",
            "properties": {
                "enabled": {"type": "boolean"}
            }
        },
        "monitoring": {
            "type": "object",
            "properties": {
                "prometheus": {
                    "type": "object",
                    "properties": {
                        "enabled": {"type": "boolean"}
                    }
                }
            }
        }
    }
}

def validate_grand_central(file_path: str) -> bool:
    """Validate Grand Central configuration file"""
    errors = []
    warnings = []
    
    try:
        with open(file_path, 'r') as f:
            config = yaml.safe_load(f)
    except yaml.YAMLError as e:
        print(f"❌ YAML parsing error: {e}")
        return False
    
    # Schema validation
    try:
        validate(instance=config, schema=GRAND_CENTRAL_SCHEMA)
    except ValidationError as e:
        errors.append(f"Schema validation failed: {e.message}")
    
    # Platform-specific validations
    
    # 1. Check for production requirements
    prod_env = next((env for env in config.get('environments', []) 
                     if env.get('name') == 'production'), None)
    
    if prod_env:
        # Production must have change management
        if not config.get('change_management', {}).get('enabled'):
            errors.append("Production environment must have change_management enabled")
        
        # Production should have deployment windows
        if 'deployment_windows' not in prod_env:
            warnings.append("Production should define deployment_windows")
        
        # Production must have multiple cells
        if 'cells' in prod_env.get('kubernetes', {}):
            if len(prod_env['kubernetes']['cells']) < 2:
                errors.append("Production must deploy to at least 2 cells for HA")
    
    # 2. Check deployment hooks
    if 'base_environment' in config:
        hooks = config['base_environment'].get('deployment_hooks', {})
        required_hooks = ['pre_deploy', 'post_deploy']
        
        for hook in required_hooks:
            if hook not in hooks:
                errors.append(f"Missing required deployment hook: {hook}")
            else:
                # Check hook scripts exist
                for hook_config in hooks[hook]:
                    if 'script' in hook_config:
                        script_path = Path(hook_config['script'])
                        if not script_path.exists():
                            errors.append(f"Hook script not found: {hook_config['script']}")
    
    # 3. Check for required integrations
    env_vars = config.get('base_environment', {}).get('env_vars', {})
    
    # Check for New Relic monitoring
    if 'NEW_RELIC_APP_NAME' not in env_vars:
        errors.append("Missing NEW_RELIC_APP_NAME environment variable")
    
    # Check for proper secret references
    for key, value in env_vars.items():
        if any(secret in key.lower() for secret in ['password', 'token', 'key', 'secret']):
            if not str(value).startswith('secret_key_ref:'):
                errors.append(f"Secret '{key}' should use secret_key_ref")
    
    # 4. Check autoscaling configuration
    for env in config.get('environments', []):
        if env.get('autoscaling', {}).get('enabled'):
            max_replicas = env['autoscaling'].get('max_replicas', 0)
            if max_replicas > 50:
                warnings.append(f"Environment {env['name']} has very high max_replicas: {max_replicas}")
    
    # 5. Check for entity platform integration
    if 'entity_synthesis' not in config:
        warnings.append("Missing entity_synthesis configuration for observability")
    
    # Print results
    if errors or warnings:
        print(f"\n🔍 Validating: {file_path}")
        
        if errors:
            print("\n❌ ERRORS:")
            for error in errors:
                print(f"  • {error}")
        
        if warnings:
            print("\n⚠️  WARNINGS:")
            for warning in warnings:
                print(f"  • {warning}")
        
        return len(errors) == 0
    else:
        print(f"✅ {file_path} - Valid")
        return True

def main():
    """Main entry point"""
    if len(sys.argv) < 2:
        print("Usage: validate-grand-central.py <grandcentral.yml>")
        sys.exit(1)
    
    all_valid = True
    for file_path in sys.argv[1:]:
        if not validate_grand_central(file_path):
            all_valid = False
    
    sys.exit(0 if all_valid else 1)

if __name__ == '__main__':
    main()