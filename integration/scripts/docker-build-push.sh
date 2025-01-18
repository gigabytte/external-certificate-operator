#!/bin/bash

# Exit immediately if a command exits with a non-zero status
set -e

# Function to display usage
usage() {
    echo "Usage: $0 -r <acr_name> -i <image_name> -t <image_tag>"
    exit 1
}

# Parse command-line arguments
while getopts "r:i:t:" opt; do
    case $opt in
        r) ACR_NAME="$OPTARG" ;;
        i) IMAGE_NAME="$OPTARG" ;;
        t) IMAGE_TAG="$OPTARG" ;;
        *) usage ;;
    esac
done

# Prompt for missing arguments
if [ -z "$ACR_NAME" ]; then
    read -p "Enter ACR name: " ACR_NAME
fi

if [ -z "$IMAGE_NAME" ]; then
    read -p "Enter image name: " IMAGE_NAME
fi

if [ -z "$IMAGE_TAG" ]; then
    read -p "Enter image tag: " IMAGE_TAG
fi

# Log in to ACR
echo "Logging in to ACR..."
az acr login --name "$ACR_NAME"

# Build the Docker image
echo "Building Docker image..."
docker build --platform=linux/amd64 -t "$IMAGE_NAME:$IMAGE_TAG" .

# Tag the Docker image
echo "Tagging Docker image..."
docker tag "$IMAGE_NAME:$IMAGE_TAG" "$ACR_NAME.azurecr.io/$IMAGE_NAME:$IMAGE_TAG"

# Push the Docker image to ACR
echo "Pushing Docker image to ACR..."
docker push "$ACR_NAME.azurecr.io/$IMAGE_NAME:$IMAGE_TAG"

echo "Docker image pushed to ACR successfully."
