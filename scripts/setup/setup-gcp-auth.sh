#!/bin/bash
# GCP Authentication Setup Script for WGO

set -e

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${GREEN}WGO GCP Authentication Setup${NC}"
echo "============================"

# Check if gcloud is installed
if ! command -v gcloud &> /dev/null; then
    echo -e "${RED}Error: gcloud CLI is not installed${NC}"
    echo "Please install it from: https://cloud.google.com/sdk/docs/install"
    exit 1
fi

# Get current auth status
echo -e "\n${YELLOW}Current authentication status:${NC}"
gcloud auth list

# Get current project
CURRENT_PROJECT=$(gcloud config get-value project 2>/dev/null)
echo -e "\n${YELLOW}Current project:${NC} ${CURRENT_PROJECT:-Not set}"

# Ask user for project ID
echo -e "\n${YELLOW}Enter your GCP project ID (or press Enter to use current):${NC}"
read -r PROJECT_ID

if [ -z "$PROJECT_ID" ]; then
    if [ -z "$CURRENT_PROJECT" ]; then
        echo -e "${RED}No project specified and no current project set${NC}"
        exit 1
    fi
    PROJECT_ID=$CURRENT_PROJECT
fi

# Set project
echo -e "\n${YELLOW}Setting project to:${NC} $PROJECT_ID"
gcloud config set project "$PROJECT_ID"

# Check if already authenticated
if gcloud auth application-default print-access-token &> /dev/null; then
    echo -e "\n${GREEN}✓ Already authenticated with Application Default Credentials${NC}"
else
    echo -e "\n${YELLOW}Setting up Application Default Credentials...${NC}"
    gcloud auth application-default login
fi

# Verify access
echo -e "\n${YELLOW}Verifying access to project...${NC}"
if gcloud projects describe "$PROJECT_ID" &> /dev/null; then
    echo -e "${GREEN}✓ Successfully accessed project $PROJECT_ID${NC}"
else
    echo -e "${RED}✗ Cannot access project $PROJECT_ID${NC}"
    echo "Please ensure you have the necessary permissions"
    exit 1
fi

# Test with WGO
echo -e "\n${YELLOW}Testing WGO GCP scanner...${NC}"
if [ -f "./wgo" ]; then
    ./wgo scan --provider gcp --project "$PROJECT_ID"
else
    echo -e "${RED}WGO binary not found. Please build it first:${NC}"
    echo "go build -o wgo ./cmd/wgo"
fi

echo -e "\n${GREEN}Setup complete!${NC}"
echo -e "You can now use: ${YELLOW}./wgo scan --provider gcp --project $PROJECT_ID${NC}"