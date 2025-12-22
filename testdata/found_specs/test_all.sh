#!/bin/bash
# Test all found specs and collect error types

cd "$(dirname "$0")/../.."

for spec in testdata/found_specs/*.toml; do
    name=$(basename "$spec" .toml)
    echo "========================================
Testing: $name"
    ./timewinder run "$spec" 2>&1 | head -20
    echo ""
done
