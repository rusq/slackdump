#!/bin/bash
set -euo pipefail

# Script to add AGPL-3.0 license headers to all Go files
# Excludes: ./cmd/slackdump/internal/golang/* (BSD-licensed)

# AGPL-3.0 header template
read -r -d '' HEADER <<'EOF' || true
// Copyright (c) 2021-2026 Rustam Gilyazov and Contributors.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

EOF

# Counter
count=0
skipped=0

# Find all Go files excluding the golang directory
while IFS= read -r file; do
    # Check if file already has AGPL header
    if head -n 5 "$file" 2>/dev/null | grep -q "GNU Affero General Public License"; then
        echo "SKIP (already has header): $file"
        ((skipped++))
        continue
    fi
    
    # Create temporary file
    temp_file=$(mktemp)
    
    # Add header and original content
    echo "$HEADER" > "$temp_file"
    cat "$file" >> "$temp_file"
    
    # Replace original file
    mv "$temp_file" "$file"
    
    echo "ADDED: $file"
    ((count++))
done < <(git ls-files "*.go" | grep -v "^cmd/slackdump/internal/golang/")

echo ""
echo "========================================="
echo "Summary:"
echo "  Headers added: $count"
echo "  Files skipped: $skipped"
echo "========================================="
echo ""
echo "Next steps:"
echo "1. Review the changes: git diff"
echo "2. Update LICENSE file to AGPL-3.0 if needed"
echo "3. Commit: git commit -am 'Change license to AGPL-3.0'"
