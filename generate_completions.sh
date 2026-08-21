#!/bin/bash
# generate_completions.sh

# 1. Create the directory
mkdir -p completions

# 2. Build a temporary binary
go build -o dotdo_temp main.go

# 3. Generate the files using the command we already built!
./dotdo_temp completion bash > completions/dotdo.bash
./dotdo_temp completion zsh > completions/dotdo.zsh

# 4. Clean up the temp binary
rm dotdo_temp

echo "● Completion files generated in ./completions/"