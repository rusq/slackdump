#!/bin/bash
# Fix missing blank line after AGPL license headers in Go files
# This ensures proper Go package documentation formatting

set -e

# Count files that will be modified
count=0

# Find all .go files (excluding vendor and generated mocks)
while IFS= read -r file; do
    # Check if file has the AGPL license header
    if ! grep -q "GNU Affero General Public License" "$file"; then
        continue
    fi
    
    # Check if there's already a blank line after the license
    # The license ends with: "// along with this program.  If not, see <https://www.gnu.org/licenses/>."
    # We need to check if the next line after that is blank
    
    # Use awk to check and fix in one pass
    awk '
    BEGIN { 
        in_license = 0
        found_end = 0
    }
    /^\/\/ Copyright/ { 
        in_license = 1 
    }
    /^\/\/ along with this program.*gnu\.org\/licenses/ {
        found_end = 1
        print
        next
    }
    found_end == 1 {
        # This is the line right after license end
        if ($0 !~ /^$/) {
            # Not a blank line, add one
            print ""
        }
        found_end = 0
        in_license = 0
    }
    { print }
    ' "$file" > "$file.tmp"
    
    # Only replace if file actually changed
    if ! cmp -s "$file" "$file.tmp"; then
        mv "$file.tmp" "$file"
        echo "Fixed: $file"
        ((count++))
    else
        rm "$file.tmp"
    fi
done < <(find . -name "*.go" -type f \
    ! -path "*/vendor/*" \
    ! -path "*/.git/*" \
    ! -path "*/mocks/mock_*/*" \
    ! -path "*/mock_*/*")

echo ""
echo "✓ Fixed $count files"
