#!/bin/bash

# Create a coverage output directory if it doesn't exist
mkdir -p coverage

# Run tests with coverage for all packages
go test ./... -coverprofile=coverage/coverage.out

# Get the coverage percentage
go tool cover -func=coverage/coverage.out

# Generate HTML coverage report
go tool cover -html=coverage/coverage.out -o coverage/coverage.html

# Display total coverage
echo "Coverage report has been generated in coverage/coverage.html"