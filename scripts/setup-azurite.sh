#!/usr/bin/env bash
#
# Setup script for Azurite (Azure Storage Emulator) local development
#
# This script:
# 1. Starts Azurite using docker-compose
# 2. Waits for Azurite to be ready
# 3. Creates the incident-reports container
# 4. Displays configuration for use with the runner
#

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

CONTAINER_NAME="incident-reports"
CONNECTION_STRING="DefaultEndpointsProtocol=http;AccountName=devstoreaccount1;AccountKey=Eby8vdM02xNOcqFlqUwJPLlmEtlCDXJ1OUzFT50uSRZ6IFsuFq2UVErCz4I6tq/K1SZFPTOtr/KBHBeksoGMGw==;BlobEndpoint=http://127.0.0.1:10000/devstoreaccount1;"

echo "Setting up Azurite for local development..."
echo

# Start Azurite
echo "Starting Azurite container..."
cd "${PROJECT_ROOT}"
docker-compose -f docker-compose.azurite.yml up -d

# Wait for Azurite to be ready
echo "Waiting for Azurite to be ready..."
MAX_ATTEMPTS=30
ATTEMPT=0
while [ $ATTEMPT -lt $MAX_ATTEMPTS ]; do
    if curl -s http://127.0.0.1:10000/devstoreaccount1?comp=list > /dev/null 2>&1; then
        echo "Azurite is ready!"
        break
    fi
    ATTEMPT=$((ATTEMPT + 1))
    if [ $ATTEMPT -eq $MAX_ATTEMPTS ]; then
        echo "Error: Azurite did not become ready in time"
        exit 1
    fi
    sleep 1
done

# Create container using curl
echo "Creating container '${CONTAINER_NAME}'..."
HTTP_DATE=$(date -u '+%a, %d %b %Y %H:%M:%S GMT' 2>/dev/null || gdate -u '+%a, %d %b %Y %H:%M:%S GMT')
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
    -X PUT "http://127.0.0.1:10000/devstoreaccount1/${CONTAINER_NAME}?restype=container" \
    -H "x-ms-date: ${HTTP_DATE}" \
    -H "x-ms-version: 2021-08-06")

if [ "$HTTP_CODE" = "201" ]; then
    echo "Container '${CONTAINER_NAME}' created successfully"
elif [ "$HTTP_CODE" = "409" ]; then
    echo "Container '${CONTAINER_NAME}' already exists"
else
    echo "Warning: Unexpected HTTP response code: ${HTTP_CODE}"
fi

echo
echo "Azurite setup complete!"
echo
echo "To use Azurite with the incident runner, set these environment variables:"
echo
echo "export AZURE_STORAGE_CONNECTION_STRING=\"${CONNECTION_STRING}\""
echo "export AZURE_STORAGE_CONTAINER=\"${CONTAINER_NAME}\""
echo
echo "To stop Azurite:"
echo "  docker-compose -f docker-compose.azurite.yml down"
echo
echo "To view Azurite logs:"
echo "  docker-compose -f docker-compose.azurite.yml logs -f"
echo
echo "To list blobs in the container:"
echo "  curl -s \"http://127.0.0.1:10000/devstoreaccount1/${CONTAINER_NAME}?restype=container&comp=list\" | xmllint --format -"
echo
