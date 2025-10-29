#!/usr/bin/env bash

set -euo pipefail

# Release automation script for opencode-bot
# Usage: ./scripts/release.sh <version>
# Example: ./scripts/release.sh v1.0.0

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Check if version argument is provided
if [ $# -eq 0 ]; then
    echo -e "${RED}Error: Version argument is required${NC}"
    echo "Usage: $0 <version>"
    echo "Example: $0 v1.0.0"
    exit 1
fi

VERSION=$1
PROJECT_NAME="opencode-bot"
REPO_NAME="github.com/cnap-app/opencode-bot"

# Validate version format (vX.Y.Z)
if ! [[ $VERSION =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo -e "${RED}Error: Invalid version format${NC}"
    echo "Version must be in format vX.Y.Z (e.g., v1.0.0)"
    exit 1
fi

echo -e "${BLUE}=== Release Script for ${PROJECT_NAME} ===${NC}"
echo "Version: ${VERSION}"
echo ""

# Check if git working directory is clean
if [[ -n $(git status -s) ]]; then
    echo -e "${RED}Error: Git working directory is not clean${NC}"
    echo "Please commit or stash your changes first"
    git status -s
    exit 1
fi

# Check if we're on main/master branch
CURRENT_BRANCH=$(git branch --show-current)
if [[ "$CURRENT_BRANCH" != "main" && "$CURRENT_BRANCH" != "master" ]]; then
    echo -e "${YELLOW}Warning: You're not on main/master branch (current: ${CURRENT_BRANCH})${NC}"
    read -p "Continue anyway? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

# Check if tag already exists
if git rev-parse "$VERSION" >/dev/null 2>&1; then
    echo -e "${RED}Error: Tag $VERSION already exists${NC}"
    exit 1
fi

# Update version in relevant files (if any)
echo -e "${YELLOW}Updating version references...${NC}"
# Add version file updates here if needed
# Example: sed -i '' "s/Version = \".*\"/Version = \"${VERSION}\"/" version.go

# Run tests
echo -e "${YELLOW}Running tests...${NC}"
if ! make test; then
    echo -e "${RED}Tests failed. Please fix before releasing.${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Tests passed${NC}"
echo ""

# Run linter
echo -e "${YELLOW}Running linter...${NC}"
if ! make lint; then
    echo -e "${RED}Linting failed. Please fix before releasing.${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Linting passed${NC}"
echo ""

# Build binaries
echo -e "${YELLOW}Building binaries...${NC}"
./scripts/build.sh -v "${VERSION#v}" --checksums
if [ $? -ne 0 ]; then
    echo -e "${RED}Build failed${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Binaries built successfully${NC}"
echo ""

# Generate changelog
echo -e "${YELLOW}Generating changelog...${NC}"
CHANGELOG_FILE="CHANGELOG-${VERSION}.md"
PREV_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "")

if [ -z "$PREV_TAG" ]; then
    echo "# Changelog for ${VERSION}" > "${CHANGELOG_FILE}"
    echo "" >> "${CHANGELOG_FILE}"
    echo "Initial release" >> "${CHANGELOG_FILE}"
else
    echo "# Changelog for ${VERSION}" > "${CHANGELOG_FILE}"
    echo "" >> "${CHANGELOG_FILE}"
    echo "## Changes since ${PREV_TAG}" >> "${CHANGELOG_FILE}"
    echo "" >> "${CHANGELOG_FILE}"
    
    # Group commits by type
    echo "### Features" >> "${CHANGELOG_FILE}"
    git log "${PREV_TAG}..HEAD" --pretty=format:"- %s (%h)" --grep="^feat" >> "${CHANGELOG_FILE}" || true
    echo "" >> "${CHANGELOG_FILE}"
    echo "" >> "${CHANGELOG_FILE}"
    
    echo "### Bug Fixes" >> "${CHANGELOG_FILE}"
    git log "${PREV_TAG}..HEAD" --pretty=format:"- %s (%h)" --grep="^fix" >> "${CHANGELOG_FILE}" || true
    echo "" >> "${CHANGELOG_FILE}"
    echo "" >> "${CHANGELOG_FILE}"
    
    echo "### Other Changes" >> "${CHANGELOG_FILE}"
    git log "${PREV_TAG}..HEAD" --pretty=format:"- %s (%h)" --grep="^feat" --grep="^fix" --invert-grep >> "${CHANGELOG_FILE}" || true
    echo "" >> "${CHANGELOG_FILE}"
fi

echo -e "${GREEN}✓ Changelog generated: ${CHANGELOG_FILE}${NC}"
echo ""

# Review changelog
echo -e "${BLUE}=== Changelog Preview ===${NC}"
cat "${CHANGELOG_FILE}"
echo ""
echo -e "${BLUE}=========================${NC}"
echo ""

read -p "Review the changelog above. Continue with release? (y/N) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Release cancelled"
    rm -f "${CHANGELOG_FILE}"
    exit 1
fi

# Create git tag
echo -e "${YELLOW}Creating git tag...${NC}"
git tag -a "$VERSION" -m "Release $VERSION"
echo -e "${GREEN}✓ Tag created${NC}"

# Push tag to remote
echo -e "${YELLOW}Pushing tag to remote...${NC}"
git push origin "$VERSION"
echo -e "${GREEN}✓ Tag pushed${NC}"
echo ""

# Build and push Docker image
echo -e "${YELLOW}Building Docker image...${NC}"
DOCKER_IMAGE="ghcr.io/cnap-app/opencode-bot"
docker build \
    --build-arg VERSION="${VERSION#v}" \
    --build-arg COMMIT="$(git rev-parse --short HEAD)" \
    --build-arg BUILD_TIME="$(date -u '+%Y-%m-%dT%H:%M:%SZ')" \
    -t "${DOCKER_IMAGE}:${VERSION}" \
    -t "${DOCKER_IMAGE}:latest" \
    .

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Docker image built${NC}"
    echo ""
    
    read -p "Push Docker image to registry? (y/N) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        echo -e "${YELLOW}Pushing Docker image...${NC}"
        docker push "${DOCKER_IMAGE}:${VERSION}"
        docker push "${DOCKER_IMAGE}:latest"
        echo -e "${GREEN}✓ Docker image pushed${NC}"
    fi
else
    echo -e "${RED}Docker build failed${NC}"
fi

echo ""
echo -e "${GREEN}=== Release ${VERSION} Complete! ===${NC}"
echo ""
echo "Next steps:"
echo "1. Go to GitHub and create a release for tag ${VERSION}"
echo "2. Upload binaries from ./dist directory"
echo "3. Copy content from ${CHANGELOG_FILE} to release notes"
echo "4. Announce the release to users"
echo ""
echo "GitHub release URL:"
echo "https://github.com/cnap-app/opencode-bot/releases/new?tag=${VERSION}"