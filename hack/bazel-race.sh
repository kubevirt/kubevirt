#!/bin/bash
#
# This script adds race detection to all go_test rules in BUILD.bazel files
# across the KubeVirt project. It adds 'race = "on"' to go_test definitions
# that don't already have a race configuration.
#

set -e

# Find all BUILD.bazel files and process them
find . -name "BUILD.bazel" -type f | while read -r file; do
    # Check if file contains go_test rules
    if grep -q "go_test(" "$file"; then
        echo "Processing $file..."

        # Create a temporary file
        temp_file=$(mktemp)

        # Process the file with awk to add race = "on" to go_test blocks
        awk '
        BEGIN {
            in_go_test = 0
            has_race = 0
            test_block = ""
            indent = ""
        }
        
        # Detect start of go_test block
        /^[[:space:]]*go_test\(/ {
            in_go_test = 1
            has_race = 0
            test_block = $0
            # Extract indentation from the go_test line
            match($0, /^[[:space:]]*/)
            indent = substr($0, RSTART, RLENGTH)
            next
        }
        
        # While inside go_test block
        in_go_test == 1 {
            test_block = test_block "\n" $0
            
            # Check if race is already configured
            if ($0 ~ /race[[:space:]]*=/) {
                has_race = 1
            }
            
            # Check for end of go_test block
            if ($0 ~ /^\)/) {
                in_go_test = 0
                
                # If no race configuration found, add it
                if (has_race == 0) {
                    # Insert race = "on" before the closing parenthesis
                    gsub(/\n\)$/, "\n" indent "    race = \"on\",\n)", test_block)
                }
                
                print test_block
                test_block = ""
                next
            }
        }
        
        # Print non-go_test lines as-is
        in_go_test == 0 {
            print $0
        }
        ' "$file" >"$temp_file"

        # Only update the file if changes were made
        if ! cmp -s "$file" "$temp_file"; then
            mv "$temp_file" "$file"
            echo "  Updated $file with race detection"
        else
            rm "$temp_file"
            echo "  No changes needed in $file"
        fi
    fi
done

echo "Completed adding race detection to go_test rules."
