#!/bin/bash

# build-go-service.sh - Script to build Go services using the generic Dockerfile

set -e

# Service configurations
declare -A SERVICES=(
    ["problems-service-go"]="8000"
    ["submissions-service-go"]="8001"
    ["plagiarism-service-go"]="8002"
    ["api-gateway"]="8080"
)

# Function to build a service
build_service() {
    local service_name=$1
    local service_port=${SERVICES[$service_name]}
    
    if [ -z "$service_port" ]; then
        echo "Error: Unknown service '$service_name'"
        echo "Available services: ${!SERVICES[@]}"
        exit 1
    fi
    
    echo "Building $service_name (port $service_port)..."
    
    # Build using the generic Dockerfile
    docker build \
        --build-arg SERVICE_NAME="$service_name" \
        --build-arg SERVICE_PORT="$service_port" \
        -f Dockerfile.go-service \
        -t "codejudge/$service_name:latest" \
        .
    
    echo "Successfully built codejudge/$service_name:latest"
}

# Function to build all services
build_all() {
    echo "Building all Go services..."
    for service in "${!SERVICES[@]}"; do
        build_service "$service"
    done
    echo "All services built successfully!"
}

# Function to push to registry
push_service() {
    local service_name=$1
    local registry=${2:-"ghcr.io/kwant-dbg"}
    
    if [ -z "${SERVICES[$service_name]}" ]; then
        echo "Error: Unknown service '$service_name'"
        exit 1
    fi
    
    local tag="$registry/codejudge-$service_name:latest"
    
    echo "Tagging and pushing $service_name to $registry..."
    docker tag "codejudge/$service_name:latest" "$tag"
    docker push "$tag"
    
    echo "Successfully pushed $tag"
}

# Function to show usage
usage() {
    cat << EOF
Usage: $0 [COMMAND] [SERVICE_NAME] [OPTIONS]

Commands:
    build [SERVICE_NAME]    Build a specific service (or 'all' for all services)
    push [SERVICE_NAME]     Push a service to registry
    help                    Show this help message

Available services: ${!SERVICES[@]}

Examples:
    $0 build problems-service-go      # Build problems service
    $0 build all                      # Build all services
    $0 push submissions-service-go    # Push submissions service to default registry
    $0 push api-gateway ghcr.io/user  # Push to custom registry

EOF
}

# Main script logic
case "${1:-}" in
    "build")
        if [ "$2" = "all" ]; then
            build_all
        elif [ -n "$2" ]; then
            build_service "$2"
        else
            echo "Error: Please specify a service name or 'all'"
            usage
            exit 1
        fi
        ;;
    "push")
        if [ -n "$2" ]; then
            push_service "$2" "$3"
        else
            echo "Error: Please specify a service name"
            usage
            exit 1
        fi
        ;;
    "help"|"--help"|"-h")
        usage
        ;;
    *)
        echo "Error: Unknown command '${1:-}'"
        usage
        exit 1
        ;;
esac