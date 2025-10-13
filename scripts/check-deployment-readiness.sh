#!/bin/bash
# Quick deployment readiness check script

echo "🔍 Checking CodeJudge deployment readiness..."

# Check if all required files exist
required_files=(
    "go.work"
    "docker-compose.yml"
    "services/go/common-go/go.mod"
    "services/go/api-gateway/Dockerfile"
    "services/go/problems-service-go/Dockerfile"
    "services/go/submissions-service-go/Dockerfile"
    "services/go/plagiarism-service-go/Dockerfile"
    "services/cpp/judge-service/Dockerfile"
)

for file in "${required_files[@]}"; do
    if [ ! -f "$file" ]; then
        echo "❌ Missing: $file"
        exit 1
    fi
done

echo "✅ All required files present"

# Check Go workspace
echo "🔧 Checking Go workspace..."
if ! go work sync; then
    echo "❌ Go workspace sync failed"
    exit 1
fi

echo "✅ Go workspace ready"

# Run tests
echo "🧪 Running tests..."
cd services/go/common-go
if ! go test ./...; then
    echo "❌ Tests failed"
    exit 1
fi
cd ../../..

echo "✅ All tests pass"

# Build check
echo "🏗️  Testing builds..."
services=("api-gateway" "problems-service-go" "submissions-service-go" "plagiarism-service-go")

for service in "${services[@]}"; do
    echo "  Building $service..."
    cd "services/go/$service"
    if ! go build -o /tmp/test-build-$$; then
        echo "❌ Build failed for $service"
        exit 1
    fi
    rm -f /tmp/test-build-$$
    cd ../../..
done

echo "✅ All services build successfully"

echo ""
echo "🎉 CodeJudge is ready for Azure deployment!"
echo ""
echo "Next steps for Azure:"
echo "  1. Create Azure Container Registry"
echo "  2. Set up GitHub repository secrets"
echo "  3. Configure Azure App Service or AKS"
echo "  4. Deploy PostgreSQL and Redis"
echo ""