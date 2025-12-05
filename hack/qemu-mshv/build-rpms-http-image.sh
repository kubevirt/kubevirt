#!/bin/bash

# Write an argument parser for REGISTRY, IMAGE_NAME, RPM_IMAGE_TAG using getopts
while getopts r:i:t:d: flag; do
    case "${flag}" in
    r) REGISTRY=${OPTARG} ;;
    i) IMAGE_NAME=${OPTARG} ;;
    t) RPM_IMAGE_TAG=${OPTARG} ;;
    d) RPMS_DIR=${OPTARG} ;;
    *)
        echo "Invalid option"
        exit 1
        ;;
    esac
done

if [ -z "$REGISTRY" ] || [ -z "$IMAGE_NAME" ] || [ -z "$RPM_IMAGE_TAG" ] || [ -z "$RPMS_DIR" ]; then
    echo "Usage: $0 -r <REGISTRY> -i <IMAGE_NAME> -t <RPM_IMAGE_TAG> -d <RPMS_DIR>"
    exit 1
fi

# Create Dockerfile for RPM distribution
cat >Dockerfile <<EOF
FROM httpd:alpine

# Copy RPMs to web directory
COPY ${RPMS_DIR} /usr/local/apache2/htdocs/

# Set proper permissions for Apache to serve files
RUN chmod -R 755 /usr/local/apache2/htdocs/ && \
    chown -R www-data:www-data /usr/local/apache2/htdocs/

EXPOSE 80
CMD ["httpd-foreground"]
EOF

# Build and tag image
docker build -t ${REGISTRY}/${IMAGE_NAME}:${RPM_IMAGE_TAG} .

echo "Built container image: ${REGISTRY}/${IMAGE_NAME}:${RPM_IMAGE_TAG}"
