#!/bin/bash

# Exit immediately if a command exits with a non-zero status
set -e

# Function to display usage
usage() {
    echo "Usage: $0 -n <namespace> -d <deployment>"
    exit 1
}

# Parse command-line arguments
while getopts "n:d:" opt; do
    case $opt in
        n) NAMESPACE="$OPTARG" ;;
        d) DEPLOYMENT="$OPTARG" ;;
        *) usage ;;
    esac
done

# Check if all required arguments are provided
if [ -z "$NAMESPACE" ] || [ -z "$DEPLOYMENT" ]; then
    usage
fi

# Perform the rollout restart
echo "Performing rollout restart for deployment '$DEPLOYMENT' in namespace '$NAMESPACE'..."
kubectl rollout restart deployment "$DEPLOYMENT" -n "$NAMESPACE"

echo "Rollout restart initiated successfully."
