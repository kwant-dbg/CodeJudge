#!/bin/bash
# Script to build and push Docker images to GitHub Container Registry
# Usage: ./scripts/build-and-push-ghcr.sh

set -e

# Configuration
REPO_OWNER="kwant-dbg"
REPO_NAME="CodeJudge"
REGISTRY="ghcr.io"
IMAGE_PREFIX="${REGISTRY}/${REPO_OWNER,,}/${REPO_NAME,,}"

# Get current git commit
COMMIT_SHA=$(git rev-parse --short HEAD)
BRANCH=$(git rev-parse --abbrev-ref HEAD)

echo "🏗️ Building and pushing CodeJudge Docker images..."
echo "Repository: ${REPO_OWNER}/${REPO_NAME}"
echo "Commit: ${COMMIT_SHA}"
echo "Branch: ${BRANCH}"
echo ""

# Services configuration
declare -A SERVICES
SERVICES[api-gateway]="api-gateway/Dockerfile ."
SERVICES[problems-service-go]="problems-service-go/Dockerfile ."
SERVICES[submissions-service-go]="submissions-service-go/Dockerfile ."
SERVICES[plagiarism-service-go]="plagiarism-service-go/Dockerfile ."
SERVICES[judge-service]="judge-service/Dockerfile judge-service"

# Build and push each service
for service in "${!SERVICES[@]}"; do
    IFS=' ' read -r dockerfile context <<< "${SERVICES[$service]}"
    
    IMAGE_NAME="${IMAGE_PREFIX}/codejudge-${service}"
    
    echo "📦 Building ${service}..."
    echo "  Dockerfile: ${dockerfile}"
    echo "  Context: ${context}"
    
    # Build with multiple tags
    docker build \
        -t "${IMAGE_NAME}:${COMMIT_SHA}" \
        -t "${IMAGE_NAME}:${BRANCH}" \
        -t "${IMAGE_NAME}:latest" \
        -f "${dockerfile}" \
        "${context}"
    
    echo "🚀 Pushing ${service}..."
    docker push "${IMAGE_NAME}:${COMMIT_SHA}"
    docker push "${IMAGE_NAME}:${BRANCH}"
    docker push "${IMAGE_NAME}:latest"
    
    echo "✅ ${service} pushed successfully"
    echo ""
done

echo "🎉 All images built and pushed successfully!"
echo ""
echo "Images available at:"
for service in "${!SERVICES[@]}"; do
    echo "  ${IMAGE_PREFIX}/codejudge-${service}:latest"
done