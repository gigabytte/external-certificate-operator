#!/bin/bash

set -e

# Define the directories
directories=(
  "integration/terraform/infra"
  "integration/terraform/operator"
  "integration/terraform/cert_manager"
)

# Function to apply Terraform
apply_terraform() {
  for dir in "${directories[@]}"; do
    echo "Applying Terraform in directory: $dir"
    pushd "$dir" > /dev/null
    terraform init
    terraform apply -auto-approve
    popd > /dev/null

    # Prompt the user to run the docker-build-push.sh script after applying infra
    if [[ "$dir" == "integration/terraform/infra" ]]; then
      read -p "Terraform apply for infra completed. Do you want to run the docker-build-push.sh script? (y/n): " answer
      if [[ "$answer" == "y" || "$answer" == "Y" ]]; then
        ./docker-build-push.sh
      else
        echo "Skipping docker-build-push.sh script."
      fi
    fi
  done
}

# Function to destroy Terraform
destroy_terraform() {
  for dir in "${directories[@]}"; do
    echo "Destroying Terraform in directory: $dir"
    pushd "$dir" > /dev/null
    terraform destroy -auto-approve
    popd > /dev/null
  done
}

# Main script
case "$1" in
  create)
    apply_terraform
    ;;
  destroy)
    destroy_terraform
    ;;
  *)
    echo "Usage: $0 {create|destroy}"
    exit 1
    ;;
esac
