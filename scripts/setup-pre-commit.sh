#!/bin/bash

set -e

echo "Setting up pre-commit hooks..."

# Check if pre-commit is installed
if ! command -v pre-commit &> /dev/null; then
    echo "pre-commit is not installed. Installing..."

    # Try different installation methods
    if command -v pip &> /dev/null; then
        pip install pre-commit
    elif command -v pip3 &> /dev/null; then
        pip3 install pre-commit
    elif command -v brew &> /dev/null; then
        brew install pre-commit
    else
        echo "Error: Could not find pip or brew to install pre-commit"
        echo "Please install pre-commit manually: https://pre-commit.com/#install"
        exit 1
    fi
fi

# Install the git hook scripts
pre-commit install

# Optionally install pre-commit for commit messages
pre-commit install --hook-type commit-msg

echo "Pre-commit hooks installed successfully!"
echo ""
echo "Available commands:"
echo "  - Run on all files: pre-commit run --all-files"
echo "  - Run with tests: pre-commit run --all-files --hook-stage manual"
echo "  - Update hooks: pre-commit autoupdate"
echo ""
echo "Hooks will run automatically on git commit."
