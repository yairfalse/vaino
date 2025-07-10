#!/bin/bash
# GCP Authentication Diagnostic Script for VAINO

set -e

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

# Usage
if [ $# -eq 0 ]; then
    echo "Usage: $0 <project-id>"
    echo "Example: $0 taskmate-46a1721"
    exit 1
fi

PROJECT_ID=$1

echo -e "${BLUE}VAINO GCP Authentication Diagnostics${NC}"
echo "===================================="
echo -e "Project: ${YELLOW}$PROJECT_ID${NC}"
echo ""

# Function to check command result
check_result() {
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓${NC} $1"
        return 0
    else
        echo -e "${RED}✗${NC} $1"
        return 1
    fi
}

# 1. Check gcloud installation
echo -e "\n${YELLOW}1. Checking gcloud CLI:${NC}"
if command -v gcloud &> /dev/null; then
    GCLOUD_VERSION=$(gcloud version --format="value(version.core)")
    echo -e "${GREEN}✓${NC} gcloud installed (version: $GCLOUD_VERSION)"
else
    echo -e "${RED}✗${NC} gcloud CLI not installed"
    echo "   Install from: https://cloud.google.com/sdk/docs/install"
    exit 1
fi

# 2. Check authentication status
echo -e "\n${YELLOW}2. Authentication Status:${NC}"
ACTIVE_ACCOUNT=$(gcloud auth list --filter=status:ACTIVE --format="value(account)" 2>/dev/null || echo "none")
if [ "$ACTIVE_ACCOUNT" != "none" ]; then
    echo -e "${GREEN}✓${NC} Active account: $ACTIVE_ACCOUNT"
else
    echo -e "${RED}✗${NC} No active gcloud account"
fi

# 3. Check Application Default Credentials
echo -e "\n${YELLOW}3. Application Default Credentials:${NC}"
ADC_PATH="$HOME/.config/gcloud/application_default_credentials.json"
if [ -f "$ADC_PATH" ]; then
    echo -e "${GREEN}✓${NC} ADC file exists at: $ADC_PATH"
    
    # Check ADC type
    ADC_TYPE=$(cat "$ADC_PATH" | grep -o '"type"[[:space:]]*:[[:space:]]*"[^"]*"' | cut -d'"' -f4 || echo "unknown")
    echo "   Type: $ADC_TYPE"
    
    # Check ADC age
    if [ "$(uname)" = "Darwin" ]; then
        ADC_AGE=$(stat -f "%Sm" -t "%Y-%m-%d %H:%M:%S" "$ADC_PATH")
    else
        ADC_AGE=$(stat -c "%y" "$ADC_PATH" | cut -d'.' -f1)
    fi
    echo "   Created/Modified: $ADC_AGE"
else
    echo -e "${RED}✗${NC} ADC file not found"
    echo "   Run: gcloud auth application-default login"
fi

# 4. Test ADC token
echo -e "\n${YELLOW}4. Testing ADC Token:${NC}"
if gcloud auth application-default print-access-token &> /dev/null; then
    echo -e "${GREEN}✓${NC} ADC token is valid"
    TOKEN_LENGTH=$(gcloud auth application-default print-access-token 2>/dev/null | wc -c)
    echo "   Token length: $TOKEN_LENGTH characters"
else
    echo -e "${RED}✗${NC} Cannot retrieve ADC token"
    echo "   Run: gcloud auth application-default login"
fi

# 5. Test project access (what vaino does)
echo -e "\n${YELLOW}5. Testing Project Access:${NC}"

# First try with gcloud
echo -n "   Testing with gcloud... "
if gcloud projects describe "$PROJECT_ID" &> /dev/null; then
    echo -e "${GREEN}✓${NC}"
    PROJECT_STATE=$(gcloud projects describe "$PROJECT_ID" --format="value(lifecycleState)" 2>/dev/null || echo "UNKNOWN")
    echo "   Project state: $PROJECT_STATE"
else
    echo -e "${RED}✗${NC}"
    echo "   Cannot access project with gcloud"
fi

# Try direct API call (mimicking vaino)
echo -n "   Testing direct API call... "
ACCESS_TOKEN=$(gcloud auth application-default print-access-token 2>/dev/null || echo "")
if [ -n "$ACCESS_TOKEN" ]; then
    API_RESPONSE=$(curl -s -H "Authorization: Bearer $ACCESS_TOKEN" \
        "https://cloudresourcemanager.googleapis.com/v1/projects/$PROJECT_ID" 2>/dev/null)
    
    if echo "$API_RESPONSE" | grep -q '"projectId"'; then
        echo -e "${GREEN}✓${NC}"
        LIFECYCLE_STATE=$(echo "$API_RESPONSE" | grep -o '"lifecycleState"[[:space:]]*:[[:space:]]*"[^"]*"' | cut -d'"' -f4 || echo "UNKNOWN")
        echo "   API access successful (state: $LIFECYCLE_STATE)"
    elif echo "$API_RESPONSE" | grep -q "403"; then
        echo -e "${RED}✗${NC}"
        echo "   Permission denied (403)"
        ERROR_MSG=$(echo "$API_RESPONSE" | grep -o '"message"[[:space:]]*:[[:space:]]*"[^"]*"' | cut -d'"' -f4 || echo "Unknown error")
        echo "   Error: $ERROR_MSG"
    else
        echo -e "${RED}✗${NC}"
        echo "   API call failed"
        echo "   Response: ${API_RESPONSE:0:100}..."
    fi
else
    echo -e "${RED}✗${NC}"
    echo "   No access token available"
fi

# 6. Check specific permissions
echo -e "\n${YELLOW}6. Checking Required Permissions:${NC}"
PERMISSIONS=(
    "resourcemanager.projects.get"
    "compute.instances.list"
    "compute.regions.list"
    "storage.buckets.list"
)

for PERM in "${PERMISSIONS[@]}"; do
    echo -n "   $PERM... "
    
    # Use testIamPermissions API
    if [ -n "$ACCESS_TOKEN" ]; then
        PERM_CHECK=$(curl -s -X POST \
            -H "Authorization: Bearer $ACCESS_TOKEN" \
            -H "Content-Type: application/json" \
            -d "{\"permissions\": [\"$PERM\"]}" \
            "https://cloudresourcemanager.googleapis.com/v1/projects/$PROJECT_ID:testIamPermissions" 2>/dev/null)
        
        if echo "$PERM_CHECK" | grep -q "\"$PERM\""; then
            echo -e "${GREEN}✓${NC}"
        else
            echo -e "${RED}✗${NC}"
        fi
    else
        echo -e "${YELLOW}?${NC} (no token)"
    fi
done

# 7. Diagnose and recommend
echo -e "\n${YELLOW}7. Diagnosis & Recommendations:${NC}"

ISSUES=0

# Check ADC
if [ ! -f "$ADC_PATH" ]; then
    echo -e "${RED}Issue:${NC} No Application Default Credentials found"
    echo -e "${BLUE}Fix:${NC} Run: gcloud auth application-default login"
    ((ISSUES++))
elif ! gcloud auth application-default print-access-token &> /dev/null; then
    echo -e "${RED}Issue:${NC} ADC token is invalid or expired"
    echo -e "${BLUE}Fix:${NC} Run: gcloud auth application-default login"
    ((ISSUES++))
fi

# Check project access
if ! gcloud projects describe "$PROJECT_ID" &> /dev/null 2>&1; then
    echo -e "${RED}Issue:${NC} Cannot access project $PROJECT_ID"
    echo -e "${BLUE}Fix:${NC} Verify:"
    echo "   1. Project ID is correct"
    echo "   2. You have access to the project"
    echo "   3. Try: gcloud config set project $PROJECT_ID"
    ((ISSUES++))
fi

# Check if using personal account with org restrictions
if [ "$ADC_TYPE" = "authorized_user" ]; then
    echo -e "${YELLOW}Note:${NC} Using personal account authentication"
    echo "   Some organizations restrict API access for personal accounts"
    echo "   Consider using a service account for production use"
fi

if [ $ISSUES -eq 0 ]; then
    echo -e "${GREEN}✓ No authentication issues detected${NC}"
    echo ""
    echo "If VAINO still fails, try:"
    echo "1. Enable verbose logging: export VAINO_LOG_LEVEL=debug"
    echo "2. Check if required APIs are enabled in your project"
    echo "3. Verify quota limits haven't been exceeded"
else
    echo -e "\n${RED}Found $ISSUES issue(s) that need to be resolved${NC}"
fi

# Additional debugging info
echo -e "\n${YELLOW}8. Additional Debug Info:${NC}"
echo "Environment variables:"
echo "   GOOGLE_CLOUD_PROJECT=${GOOGLE_CLOUD_PROJECT:-<not set>}"
echo "   GOOGLE_APPLICATION_CREDENTIALS=${GOOGLE_APPLICATION_CREDENTIALS:-<not set>}"

# Test with vaino if available
if [ -f "./vaino" ]; then
    echo -e "\n${YELLOW}9. Testing with VAINO:${NC}"
    echo "Running: ./vaino scan --provider gcp --project $PROJECT_ID"
    ./vaino scan --provider gcp --project "$PROJECT_ID" 2>&1 | head -20
fi

echo -e "\n${BLUE}Diagnostic complete!${NC}"