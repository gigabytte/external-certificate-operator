#!/bin/bash

# Exit immediately if a command exits with a non-zero status
set -e

# Generate a private key
openssl genpkey -algorithm RSA -out ca.key -pkeyopt rsa_keygen_bits:2048

# Generate a self-signed certificate
openssl req -x509 -new -nodes -key ca.key -sha256 -days 365 -out ca.crt -subj "/CN=My CA"
