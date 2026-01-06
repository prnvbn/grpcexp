#!/bin/bash
set -euo pipefail

# Check for required dependencies
for cmd in curl grep cut tr xargs; do
    if ! command -v $cmd &>/dev/null; then
        echo "Error: '$cmd' is required but not installed. Please install it and try again."
        exit 1
    fi
done

# Determine platform
platform=$(uname -ms)
case $platform in
    'Darwin x86_64') target_platform=darwin-amd64 ;;
    'Darwin arm64') target_platform=darwin-arm64 ;;
    'Linux aarch64' | 'Linux arm64') target_platform=linux-arm64 ;;
    'Linux x86_64') target_platform=linux-amd64 ;;
    *)
        echo "Unsupported platform: ${platform}"
        echo "Please open an issue: https://github.com/prnvbn/grpcexp/issues/new"
        exit 1
    ;;
esac

echo "Detected platform: $platform -> Target binary: $target_platform"
echo "Fetching release information..."

# Fetch the latest release info with error handling
api_response=$(curl -sf -w "\n%{http_code}" https://api.github.com/repos/prnvbn/grpcexp/releases/latest 2>/dev/null || echo "")

if [[ -z "$api_response" ]]; then
    echo "Error: Failed to connect to GitHub API. Please check your internet connection."
    exit 1
fi

# Extract HTTP status code (last line)
http_code=$(echo "$api_response" | tail -n1)
api_body=$(echo "$api_response" | sed '$d')

# Check if we got an error response
if [[ "$http_code" != "200" ]]; then
    echo "Error: GitHub API returned status code $http_code"
    if echo "$api_body" | grep -q "<html"; then
        echo "Received HTML error page instead of JSON. This may be a temporary issue."
    fi
    exit 1
fi

# Check if response is HTML (error page) instead of JSON
if echo "$api_body" | grep -q "^<html"; then
    echo "Error: Received HTML error page instead of JSON from GitHub API."
    echo "This may be a temporary issue. Please try again later."
    exit 1
fi

# Parse download URL - try jq first, fallback to grep/cut
if command -v jq &>/dev/null; then
    download_url=$(echo "$api_body" | jq -r ".assets[] | select(.name | contains(\"$target_platform\")) | .browser_download_url" | head -n1)
else
    # Fallback: parse JSON with grep/cut (more robust)
    download_url=$(echo "$api_body" | grep -o "\"browser_download_url\":\"[^\"]*$target_platform[^\"]*\"" | cut -d '"' -f 4 | head -n1)
fi

if [[ -z "$download_url" ]] || [[ "$download_url" == "null" ]]; then
    echo "Error: Could not find a compatible binary for $target_platform."
    echo "Available assets:"
    if command -v jq &>/dev/null; then
        echo "$api_body" | jq -r ".assets[].name" || true
    else
        echo "$api_body" | grep -o "\"name\":\"[^\"]*\"" | cut -d '"' -f 4 || true
    fi
    exit 1
fi

echo "Downloading binary from: $download_url"

# Download with error handling and validation
if ! curl -sfL "$download_url" -o grpcexp; then
    echo "Error: Failed to download binary. Please check your internet connection."
    exit 1
fi

# Validate downloaded file is not HTML/error page
if file grpcexp 2>/dev/null | grep -qi "html\|text"; then
    echo "Error: Downloaded file appears to be HTML/text instead of a binary."
    echo "This may indicate a download error. First few lines:"
    head -n3 grpcexp
    rm -f grpcexp
    exit 1
fi

# Check file size (should be > 0 and reasonable)
file_size=$(stat -f%z grpcexp 2>/dev/null || stat -c%s grpcexp 2>/dev/null || echo "0")
if [[ "$file_size" -lt 1000 ]]; then
    echo "Error: Downloaded file is too small ($file_size bytes). This may indicate a download error."
    rm -f grpcexp
    exit 1
fi

chmod +x grpcexp

echo "-------------------------------------------------------------------"
echo "✅ grpcexp has been downloaded to: $(pwd)/grpcexp"
echo ""

# Ask where to move the binary (loop until valid input)
while true; do
    echo "Where would you like to install grpcexp?"
    echo "1) /usr/local/bin (system-wide, requires sudo)"
    echo "2) ~/.local/bin (user only, no sudo needed)"
    echo "3) Keep it in the current directory"
    read -p "Enter choice (1/2/3): " choice

    case "$choice" in
        1)
            sudo mv grpcexp /usr/local/bin/
            echo "✅ grpcexp has been installed globally! You can now run 'grpcexp' from anywhere."
            break
        ;;
        2)
            mkdir -p "$HOME/.local/bin"
            mv grpcexp "$HOME/.local/bin/"
            echo "✅ grpcexp has been installed to ~/.local/bin."
            
            # Check if ~/.local/bin is in PATH
            if [[ ! "$PATH" =~ (^|:)"$HOME/.local/bin"(:|$) ]]; then
                echo "ℹ️  ~/.local/bin is not in your PATH."
                echo "   Add this to your shell profile (e.g., ~/.bashrc or ~/.zshrc):"
                echo '   export PATH="$HOME/.local/bin:$PATH"'
            fi
            break
        ;;
        3)
            echo "ℹ️  grpcexp will remain in the current directory."
            echo "   You can move it manually later if needed."
            break
        ;;
        *)
            echo "❌ Invalid choice. Please enter 1, 2, or 3."
        ;;
    esac
done

echo ""
echo "ℹ️  To enable auto-completion, visit:"
echo "   https://github.com/prnvbn/grpcexp/tree/main?tab=readme-ov-file#enabling-command-autocompletion"
echo "-------------------------------------------------------------------"