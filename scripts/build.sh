#!/usr/bin/env bash

set -euo pipefail

# Build script for opencode-bot with cross-compilation support
# Usage: ./scripts/build.sh [options]
# Options:
#   -v, --version    Version number (default: dev)
#   -o, --output     Output directory (default: ./dist)
#   -p, --platforms  Platforms to build (default: linux/amd64,darwin/amd64,windows/amd64)
#   -c, --compress   Compress binaries with UPX
#   --checksums      Generate checksums

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Default values
VERSION="${VERSION:-dev}"
OUTPUT_DIR="${OUTPUT_DIR:-./dist}"
PLATFORMS="${PLATFORMS:-linux/amd64,darwin/amd64,darwin/arm64,windows/amd64}"
COMPRESS=false
GENERATE_CHECKSUMS=false

# Build information
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(date -u '+%Y-%m-%dT%H:%M:%SZ')
PROJECT_NAME="opencode-bot"
MAIN_PATH="./cmd/opencode-bot"

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -v|--version)
            VERSION="$2"
            shift 2
            ;;
        -o|--output)
            OUTPUT_DIR="$2"
            shift 2
            ;;
        -p|--platforms)
            PLATFORMS="$2"
            shift 2
            ;;
        -c|--compress)
            COMPRESS=true
            shift
            ;;
        --checksums)
            GENERATE_CHECKSUMS=true
            shift
            ;;
        -h|--help)
            echo "Usage: $0 [options]"
            echo "Options:"
            echo "  -v, --version     Version number (default: dev)"
            echo "  -o, --output      Output directory (default: ./dist)"
            echo "  -p, --platforms   Platforms to build (comma-separated)"
            echo "  -c, --compress    Compress binaries with UPX"
            echo "  --checksums       Generate SHA256 checksums"
            echo ""
            echo "Example:"
            echo "  $0 -v 1.0.0 -p linux/amd64,darwin/amd64 --checksums"
            exit 0
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            exit 1
            ;;
    esac
done

# Print build information
echo -e "${GREEN}Building ${PROJECT_NAME}${NC}"
echo "Version: ${VERSION}"
echo "Commit: ${COMMIT}"
echo "Build Time: ${BUILD_TIME}"
echo "Output: ${OUTPUT_DIR}"
echo ""

# Create output directory
mkdir -p "${OUTPUT_DIR}"

# Build for each platform
IFS=',' read -ra PLATFORM_ARRAY <<< "$PLATFORMS"
for platform in "${PLATFORM_ARRAY[@]}"; do
    IFS='/' read -r GOOS GOARCH <<< "$platform"
    
    output_name="${PROJECT_NAME}-${VERSION}-${GOOS}-${GOARCH}"
    if [ "$GOOS" = "windows" ]; then
        output_name="${output_name}.exe"
    fi
    
    output_path="${OUTPUT_DIR}/${output_name}"
    
    echo -e "${YELLOW}Building for ${GOOS}/${GOARCH}...${NC}"
    
    # Build flags
    LDFLAGS="-w -s"
    LDFLAGS="${LDFLAGS} -X main.Version=${VERSION}"
    LDFLAGS="${LDFLAGS} -X main.Commit=${COMMIT}"
    LDFLAGS="${LDFLAGS} -X main.BuildTime=${BUILD_TIME}"
    
    # Build command
    env GOOS="${GOOS}" GOARCH="${GOARCH}" CGO_ENABLED=0 go build \
        -ldflags="${LDFLAGS}" \
        -trimpath \
        -o "${output_path}" \
        "${MAIN_PATH}"
    
    if [ $? -eq 0 ]; then
        file_size=$(du -h "${output_path}" | cut -f1)
        echo -e "${GREEN}✓ Built ${output_name} (${file_size})${NC}"
        
        # Compress with UPX if requested
        if [ "$COMPRESS" = true ] && command -v upx &> /dev/null; then
            echo -e "${YELLOW}Compressing ${output_name}...${NC}"
            upx --best --lzma "${output_path}" 2>/dev/null || upx --best "${output_path}"
            compressed_size=$(du -h "${output_path}" | cut -f1)
            echo -e "${GREEN}✓ Compressed to ${compressed_size}${NC}"
        fi
    else
        echo -e "${RED}✗ Failed to build for ${GOOS}/${GOARCH}${NC}"
        exit 1
    fi
    
    echo ""
done

# Generate checksums if requested
if [ "$GENERATE_CHECKSUMS" = true ]; then
    echo -e "${YELLOW}Generating checksums...${NC}"
    checksum_file="${OUTPUT_DIR}/checksums.txt"
    
    cd "${OUTPUT_DIR}"
    if command -v shasum &> /dev/null; then
        shasum -a 256 ${PROJECT_NAME}-* > checksums.txt
    elif command -v sha256sum &> /dev/null; then
        sha256sum ${PROJECT_NAME}-* > checksums.txt
    else
        echo -e "${RED}Warning: Neither shasum nor sha256sum found${NC}"
    fi
    cd - > /dev/null
    
    if [ -f "${checksum_file}" ]; then
        echo -e "${GREEN}✓ Checksums saved to ${checksum_file}${NC}"
    fi
fi

echo ""
echo -e "${GREEN}Build complete!${NC}"
echo "Binaries are available in: ${OUTPUT_DIR}"