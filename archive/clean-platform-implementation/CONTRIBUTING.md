# Contributing to Platform

Thank you for your interest in contributing to the Platform project! This document provides guidelines and instructions for contributing.

## Code of Conduct

By participating in this project, you agree to abide by our Code of Conduct:
- Be respectful and inclusive
- Welcome newcomers and help them get started
- Focus on what is best for the community
- Show empathy towards other community members

## How to Contribute

### Reporting Issues

Before creating an issue, please check if it already exists. When creating a new issue:

1. Use a clear and descriptive title
2. Provide a step-by-step description of the issue
3. Include any relevant logs or error messages
4. Describe the expected behavior
5. List your environment details (OS, versions, etc.)

### Suggesting Enhancements

Enhancement suggestions are welcome! Please:

1. Use a clear and descriptive title
2. Provide a detailed description of the proposed enhancement
3. Explain why this enhancement would be useful
4. List any alternative solutions you've considered

### Pull Requests

1. **Fork the repository** and create your branch from `develop`
2. **Follow the coding standards** (see below)
3. **Write tests** for your changes
4. **Update documentation** as needed
5. **Run the test suite** and ensure all tests pass
6. **Run pre-commit hooks** before committing
7. **Submit a pull request** with a clear description

## Development Setup

### Prerequisites

```bash
# Install required tools
brew install python@3.9 terraform kubectl aws-cli

# Install pre-commit
pip install pre-commit
pre-commit install
```

### Local Development

1. **Clone your fork**
   ```bash
   git clone https://github.com/YOUR_USERNAME/platform.git
   cd platform
   ```

2. **Create a virtual environment**
   ```bash
   python -m venv venv
   source venv/bin/activate  # On Windows: venv\Scripts\activate
   ```

3. **Install dependencies**
   ```bash
   pip install -r requirements-dev.txt
   ```

4. **Set up pre-commit hooks**
   ```bash
   pre-commit install
   pre-commit run --all-files  # Run on all files to verify setup
   ```

## Coding Standards

### Python

- Follow [PEP 8](https://www.python.org/dev/peps/pep-0008/)
- Use [Black](https://black.readthedocs.io/) for formatting (line length: 100)
- Use [isort](https://pycqa.github.io/isort/) for import sorting
- Write docstrings for all public functions and classes
- Use type hints where appropriate

Example:
```python
from typing import Dict, List, Optional

import requests
from flask import Flask, jsonify


def process_data(
    data: Dict[str, Any], 
    options: Optional[Dict[str, Any]] = None
) -> Dict[str, Any]:
    """
    Process input data according to specified options.
    
    Args:
        data: Input data to process
        options: Optional processing options
        
    Returns:
        Processed data dictionary
        
    Raises:
        ValueError: If data format is invalid
    """
    if not isinstance(data, dict):
        raise ValueError("Data must be a dictionary")
    
    # Processing logic here
    return {"processed": True, "data": data}
```

### Shell Scripts

- Use `#!/bin/bash` shebang
- Set `set -euo pipefail` at the beginning
- Use meaningful variable names
- Quote variables: `"${var}"`
- Check command existence before use

### Terraform

- Use consistent formatting (`terraform fmt`)
- Group related resources in modules
- Use meaningful resource names
- Always specify resource tags
- Document variables and outputs

### Kubernetes

- Use consistent label naming
- Always specify resource requests and limits
- Include health checks
- Use namespaces for isolation
- Follow security best practices

## Testing

### Running Tests

```bash
# Run all tests
make test

# Run specific test types
make test-unit
make test-integration
make test-e2e

# Run tests with coverage
pytest --cov=services --cov-report=html
```

### Writing Tests

- Write tests for all new functionality
- Maintain test coverage above 80%
- Use descriptive test names
- Test both success and failure cases
- Mock external dependencies

Example:
```python
import pytest
from unittest.mock import Mock, patch

from services.data_collector.app import process_request


class TestDataCollector:
    """Test cases for data collector service"""
    
    def test_process_request_success(self):
        """Test successful request processing"""
        mock_data = {"source": "test", "value": 42}
        
        result = process_request(mock_data)
        
        assert result["status"] == "success"
        assert result["id"] is not None
    
    def test_process_request_invalid_data(self):
        """Test request processing with invalid data"""
        with pytest.raises(ValueError):
            process_request({"invalid": "data"})
    
    @patch("services.data_collector.app.redis_client")
    def test_process_request_redis_failure(self, mock_redis):
        """Test request processing when Redis fails"""
        mock_redis.set.side_effect = Exception("Redis connection failed")
        
        with pytest.raises(Exception):
            process_request({"source": "test", "value": 42})
```

## Documentation

- Update README.md for significant changes
- Document all API endpoints
- Include examples in documentation
- Keep documentation up to date with code
- Use clear and concise language

### API Documentation Format

```yaml
endpoint: /api/v1/collect
method: POST
description: Collect data from various sources
parameters:
  - name: source
    type: string
    required: true
    description: Data source identifier
  - name: metrics
    type: object
    required: true
    description: Metrics data
response:
  200:
    description: Success
    schema:
      id: string
      status: string
  400:
    description: Invalid request
    schema:
      error: string
      details: object
```

## Commit Messages

Follow the [Conventional Commits](https://www.conventionalcommits.org/) specification:

```
<type>(<scope>): <subject>

<body>

<footer>
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, etc.)
- `refactor`: Code refactoring
- `test`: Test additions or modifications
- `chore`: Maintenance tasks
- `perf`: Performance improvements

Examples:
```
feat(api): add data validation endpoint

Add new endpoint for validating data before processing.
This helps catch errors early and provides better user feedback.

Closes #123
```

```
fix(collector): handle Redis connection timeout

Implement retry logic for Redis connections to handle
temporary network issues gracefully.
```

## Release Process

1. Create a release branch from `develop`
2. Update version numbers
3. Update CHANGELOG.md
4. Create pull request to `main`
5. After review and approval, merge to `main`
6. Tag the release
7. Deploy to production

## Getting Help

- Check the [documentation](docs/)
- Search existing issues
- Ask in the #platform-dev Slack channel
- Contact the maintainers

## Recognition

Contributors will be recognized in:
- CONTRIBUTORS.md file
- Release notes
- Project documentation

Thank you for contributing to Platform! 🎉