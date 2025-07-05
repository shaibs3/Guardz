#!/bin/bash

# Test script for Guardz URL service
# Sends 100 POST requests with paths and URLs, then verifies with GET requests

BASE_URL="http://localhost:8080"
TOTAL_REQUESTS=100
LOG_FILE="test_results.log"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Initialize counters
POST_SUCCESS=0
POST_FAILED=0
GET_SUCCESS=0
GET_FAILED=0
VERIFICATION_SUCCESS=0
VERIFICATION_FAILED=0

# Arrays to store test data (compatible with older bash)
PATHS=()
URLS=()

# Function to log messages
log() {
    echo -e "$1" | tee -a "$LOG_FILE"
}

# Function to generate random path
generate_random_path() {
    echo "path_$(date +%s)_$RANDOM"
}

# Function to generate random URLs
generate_random_urls() {
    local count=$((RANDOM % 5 + 1))  # 1-5 URLs per path
    local urls="["
    
    for ((i=1; i<=count; i++)); do
        if [ $i -gt 1 ]; then
            urls="$urls,"
        fi
        urls="$urls\"https://example$RANDOM.com/page$RANDOM\""
    done
    
    urls="$urls]"
    echo "$urls"
}

# Function to send POST request
send_post_request() {
    local path="$1"
    local urls="$2"
    local request_id="$3"
    
    log "${BLUE}[$request_id] POST /$path${NC}"
    log "URLs: $urls"
    
    response=$(curl -s -w "\n%{http_code}" -X POST \
        -H "Content-Type: application/json" \
        -d "{\"urls\": $urls}" \
        "$BASE_URL/$path")
    
    http_code=$(echo "$response" | tail -n1)
    response_body=$(echo "$response" | sed '$d')
    
    if [ "$http_code" = "200" ] || [ "$http_code" = "201" ]; then
        log "${GREEN}✓ POST /$path successful (HTTP $http_code)${NC}"
        ((POST_SUCCESS++))
        return 0
    else
        log "${RED}✗ POST /$path failed (HTTP $http_code): $response_body${NC}"
        ((POST_FAILED++))
        return 1
    fi
}

# Function to send GET request and verify
send_get_request() {
    local path="$1"
    local expected_urls="$2"
    local request_id="$3"
    
    log "${BLUE}[$request_id] GET /$path${NC}"
    
    response=$(curl -s -w "\n%{http_code}" -X GET "$BASE_URL/$path")
    http_code=$(echo "$response" | tail -n1)
    response_body=$(echo "$response" | sed '$d')
    
    if [ "$http_code" = "200" ]; then
        log "${GREEN}✓ GET /$path successful (HTTP $http_code)${NC}"
        ((GET_SUCCESS++))
        
        # Verify the response contains the expected URLs
        if verify_urls_in_response "$response_body" "$expected_urls"; then
            log "${GREEN}✓ Verification successful for /$path${NC}"
            ((VERIFICATION_SUCCESS++))
            return 0
        else
            log "${RED}✗ Verification failed for /$path${NC}"
            log "Expected URLs: $expected_urls"
            log "Response: $response_body"
            ((VERIFICATION_FAILED++))
            return 1
        fi
    else
        log "${RED}✗ GET /$path failed (HTTP $http_code): $response_body${NC}"
        ((GET_FAILED++))
        return 1
    fi
}

# Function to verify URLs in response
verify_urls_in_response() {
    local response="$1"
    local expected_urls="$2"
    
    # Extract URLs from the expected JSON array
    expected_urls_clean=$(echo "$expected_urls" | sed 's/\[//g' | sed 's/\]//g' | sed 's/"//g')
    
    # Check if response contains each expected URL
    for url in $(echo "$expected_urls_clean" | tr ',' ' '); do
        if ! echo "$response" | grep -q "$url"; then
            return 1
        fi
    done
    
    return 0
}

# Function to print summary
print_summary() {
    log "\n${YELLOW}=== TEST SUMMARY ===${NC}"
    log "Total requests: $TOTAL_REQUESTS"
    log "${GREEN}POST Success: $POST_SUCCESS${NC}"
    log "${RED}POST Failed: $POST_FAILED${NC}"
    log "${GREEN}GET Success: $GET_SUCCESS${NC}"
    log "${RED}GET Failed: $GET_FAILED${NC}"
    log "${GREEN}Verification Success: $VERIFICATION_SUCCESS${NC}"
    log "${RED}Verification Failed: $VERIFICATION_FAILED${NC}"
    
    local success_rate=$(( (POST_SUCCESS + GET_SUCCESS) * 100 / (TOTAL_REQUESTS * 2) ))
    local verification_rate=$(( VERIFICATION_SUCCESS * 100 / TOTAL_REQUESTS ))
    
    log "\n${YELLOW}Success Rate: ${success_rate}%${NC}"
    log "${YELLOW}Verification Rate: ${verification_rate}%${NC}"
    
    if [ $VERIFICATION_FAILED -eq 0 ]; then
        log "\n${GREEN}🎉 ALL TESTS PASSED! 🎉${NC}"
        exit 0
    else
        log "\n${RED}❌ SOME TESTS FAILED! ❌${NC}"
        exit 1
    fi
}

# Function to check if service is running
check_service() {
    log "${YELLOW}Checking if service is running at $BASE_URL...${NC}"
    
    if curl -s "$BASE_URL" > /dev/null 2>&1; then
        log "${GREEN}✓ Service is running${NC}"
        return 0
    else
        log "${RED}✗ Service is not running at $BASE_URL${NC}"
        log "Please start the service first with: make compose-up"
        exit 1
    fi
}

# Main execution
main() {
    # Clear log file
    > "$LOG_FILE"
    
    log "${YELLOW}=== Guardz URL Service Test Script ===${NC}"
    log "Base URL: $BASE_URL"
    log "Total requests: $TOTAL_REQUESTS"
    log "Log file: $LOG_FILE"
    log ""
    
    # Check if service is running
    check_service
    
    log "${YELLOW}=== Phase 1: Sending POST requests ===${NC}"
    
    # Send POST requests
    for ((i=1; i<=TOTAL_REQUESTS; i++)); do
        path=$(generate_random_path)
        urls=$(generate_random_urls)
        
        # Store for verification (using parallel arrays)
        PATHS[$i]="$path"
        URLS[$i]="$urls"
        
        if send_post_request "$path" "$urls" "$i"; then
            # Small delay to avoid overwhelming the service
            sleep 0.1
        fi
    done
    
    log "\n${YELLOW}=== Phase 2: Sending GET requests and verifying ===${NC}"
    
    # Send GET requests and verify
    for ((i=1; i<=TOTAL_REQUESTS; i++)); do
        path="${PATHS[$i]}"
        expected_urls="${URLS[$i]}"
        
        send_get_request "$path" "$expected_urls" "$i"
        
        # Small delay
        sleep 0.1
    done
    
    # Print summary
    print_summary
}

# Handle script interruption
trap 'log "\n${RED}Script interrupted by user${NC}"; print_summary; exit 1' INT

# Run main function
main "$@" 