#!/bin/bash

# MangaHub API Test Script
# Comprehensive testing of all HTTP REST API endpoints

set -e

BASE_URL="http://localhost:8080"
API_V1="$BASE_URL/api/v1"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test counters
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# Function to log test results
log_test() {
    local test_name="$1"
    local status="$2"
    local response="$3"
    
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    
    if [ "$status" = "PASS" ]; then
        PASSED_TESTS=$((PASSED_TESTS + 1))
        echo -e "${GREEN}✓ PASS${NC} $test_name" >&2
    elif [ "$status" = "FAIL" ]; then
        FAILED_TESTS=$((FAILED_TESTS + 1))
        echo -e "${RED}✗ FAIL${NC} $test_name" >&2
        if [ -n "$response" ]; then
            echo -e "  ${YELLOW}Response:${NC} $response" >&2
        fi
    else
        echo -e "${YELLOW}⚠ SKIP${NC} $test_name" >&2
    fi
}

# Function to make API call and check response
api_test() {
    local method="$1"
    local endpoint="$2" 
    local expected_status="$3"
    local data="$4"
    local headers="$5"
    local test_name="$6"
    
    local curl_cmd="curl -s -w '%{http_code}' -X $method"
    
    if [ -n "$headers" ]; then
        curl_cmd="$curl_cmd -H '$headers'"
    fi
    
    if [ -n "$data" ]; then
        curl_cmd="$curl_cmd -d '$data' -H 'Content-Type: application/json'"
    fi
    
    curl_cmd="$curl_cmd $endpoint"
    
    # Execute curl and capture response
    local response
    response=$(eval $curl_cmd 2>/dev/null)
    
    if [ $? -ne 0 ]; then
        log_test "$test_name" "FAIL" "Connection failed"
        return 1
    fi
    
    # Extract status code (last 3 characters)
    local status_code="${response: -3}"
    local body="${response%???}"
    
    if [ "$status_code" = "$expected_status" ]; then
        log_test "$test_name" "PASS" 
        echo "$body"
        return 0
    else
        log_test "$test_name" "FAIL" "Expected $expected_status, got $status_code - $body"
        return 1
    fi
}

echo -e "${BLUE}🚀 Starting MangaHub API Test Suite${NC}"
echo "Testing API at: $BASE_URL"
echo "=================================================="

# Wait for server to be ready
echo "Waiting for server to start..."
for i in {1..10}; do
    if curl -s "$BASE_URL/health" > /dev/null 2>&1; then
        echo "Server is ready!"
        break
    fi
    if [ $i -eq 10 ]; then
        echo -e "${RED}❌ Server not responding after 10 seconds${NC}"
        exit 1
    fi
    sleep 1
done

# Global variables for storing tokens and IDs
AUTH_TOKEN=""
MANGA_ID=""
COMMENT_ID=""
USER_ID=""

echo -e "\n${BLUE}📋 Testing Health and Info Endpoints${NC}"

# Test health endpoint
api_test "GET" "$BASE_URL/health" "200" "" "" "Health Check"

echo -e "\n${BLUE}🔐 Testing Authentication Endpoints${NC}"

# Test user registration (unique username)
echo -e "\n--- User Registration ---"
TEST_USERNAME="testuser_$(date +%s)"
REGISTER_RESPONSE=$(api_test "POST" "$API_V1/auth/register" "201" \
    "{\"username\":\"$TEST_USERNAME\",\"password\":\"password123\"}" \
    "" "User Registration")

if [ $? -eq 0 ]; then
    # Extract user ID from response
    USER_ID=$(echo "$REGISTER_RESPONSE" | python3 -c 'import sys, json; print(json.load(sys.stdin)["data"]["user"]["id"])')
    echo "Registered user ID: $USER_ID"
fi

# Test user login
echo -e "\n--- User Login ---"
LOGIN_RESPONSE=$(api_test "POST" "$API_V1/auth/login" "200" \
    "{\"username\":\"$TEST_USERNAME\",\"password\":\"password123\"}" \
    "" "User Login")

if [ $? -eq 0 ]; then
    # Extract token from response
    AUTH_TOKEN=$(echo "$LOGIN_RESPONSE" | python3 -c 'import sys, json; print(json.load(sys.stdin)["data"]["token"])')
    echo "Auth token obtained: ${AUTH_TOKEN:0:20}..."
fi

# Test login with wrong credentials
api_test "POST" "$API_V1/auth/login" "401" \
    '{"username":"testuser123","password":"wrongpassword"}' \
    "" "Invalid Login" > /dev/null

echo -e "\n${BLUE}📚 Testing Manga Endpoints${NC}"

# Test manga list (public)
echo -e "\n--- Manga Listing ---"
api_test "GET" "$API_V1/manga" "200" "" "" "List All Manga" > /dev/null

# Test manga search (public)
echo -e "\n--- Manga Search ---"
api_test "GET" "$API_V1/manga/search?q=naruto" "200" "" "" "Search Manga" > /dev/null

if [ -n "$AUTH_TOKEN" ]; then
    # Test manga creation (protected)
    echo -e "\n--- Manga Creation ---"
    # Use admin token for admin-only create
    ADMIN_TOKEN=$(api_test "POST" "$API_V1/auth/login" "200" \
        '{"username":"admin","password":"admin123"}' \
        "" "Admin Login" | python3 -c 'import sys, json; print(json.load(sys.stdin)["data"]["token"])')
    CREATE_MANGA_RESPONSE=$(api_test "POST" "$API_V1/manga" "201" \
        '{"title":"Test Manga","description":"A test manga for API testing","cover_url":"https://example.com/cover.jpg","status":"ongoing"}' \
        "Authorization: Bearer $ADMIN_TOKEN" "Create Manga")
    
    if [ $? -eq 0 ]; then
        # Extract manga ID from response
        MANGA_ID=$(echo "$CREATE_MANGA_RESPONSE" | python3 -c 'import sys, json; print(json.load(sys.stdin)["data"]["id"])')
        echo "Created manga ID: $MANGA_ID"
        
        # Test get specific manga (public)
        echo -e "\n--- Get Specific Manga ---"
        api_test "GET" "$API_V1/manga/$MANGA_ID" "200" "" "" "Get Manga by ID" > /dev/null
        
        # Test manga update (protected)
        echo -e "\n--- Manga Update ---"
        api_test "PUT" "$API_V1/manga/$MANGA_ID" "200" \
            '{"title":"Updated Test Manga","description":"Updated description"}' \
            "Authorization: Bearer $AUTH_TOKEN" "Update Manga" > /dev/null
    fi
fi

echo -e "\n${BLUE}💬 Testing Comment Endpoints${NC}"

if [ -n "$AUTH_TOKEN" ] && [ -n "$MANGA_ID" ]; then
    # Test comment creation (protected)
    echo -e "\n--- Comment Creation ---"
    CREATE_COMMENT_RESPONSE=$(api_test "POST" "$API_V1/manga/$MANGA_ID/comments" "201" \
        '{"content":"This is a test comment for the API testing!"}' \
        "Authorization: Bearer $AUTH_TOKEN" "Create Comment")
    
    if [ $? -eq 0 ]; then
        # Extract comment ID from response
        COMMENT_ID=$(echo "$CREATE_COMMENT_RESPONSE" | python3 -c 'import sys, json; print(json.load(sys.stdin)["data"]["id"])')
        echo "Created comment ID: $COMMENT_ID"
        
        # Test list comments (public)
        echo -e "\n--- Comment Listing ---"
        api_test "GET" "$API_V1/manga/$MANGA_ID/comments" "200" "" "" "List Comments" > /dev/null
        
        # Test like comment (protected)
        echo -e "\n--- Comment Like ---"
        api_test "POST" "$API_V1/manga/$MANGA_ID/comments/$COMMENT_ID/like" "200" "" \
            "Authorization: Bearer $AUTH_TOKEN" "Like Comment" > /dev/null
    fi
fi

echo -e "\n${BLUE}📊 Testing Activity Endpoints${NC}"

# Test global activity feed (public)
echo -e "\n--- Global Activity Feed ---"
api_test "GET" "$API_V1/activity/global" "200" "" "" "Global Activity Feed" > /dev/null

if [ -n "$MANGA_ID" ]; then
    # Test manga activity feed (public)
    echo -e "\n--- Manga Activity Feed ---"
    api_test "GET" "$API_V1/activity/manga/$MANGA_ID" "200" "" "" "Manga Activity Feed" > /dev/null
fi

if [ -n "$AUTH_TOKEN" ] && [ -n "$USER_ID" ]; then
    # Test user activity feed (protected)
    echo -e "\n--- User Activity Feed ---"
    api_test "GET" "$API_V1/activity/user/$USER_ID" "200" "" \
        "Authorization: Bearer $AUTH_TOKEN" "User Activity Feed" > /dev/null
fi

echo -e "\n${BLUE}📈 Testing Stats Endpoints${NC}"

# Test top manga stats (public)
echo -e "\n--- Top Manga Stats ---"
api_test "GET" "$API_V1/stats/top?limit=5" "200" "" "" "Top Manga Stats" > /dev/null

echo -e "\n${BLUE}🔒 Testing Protected Route Security${NC}"

# Test accessing protected routes without token
echo -e "\n--- Security Tests ---"
api_test "POST" "$API_V1/manga" "401" \
    '{"title":"Unauthorized Test"}' \
    "" "Create Manga Without Auth" > /dev/null

api_test "POST" "$API_V1/manga/test-id/comments" "401" \
    '{"content":"Unauthorized comment"}' \
    "" "Create Comment Without Auth" > /dev/null

echo -e "\n${BLUE}🧹 Cleanup${NC}"

if [ -n "$AUTH_TOKEN" ] && [ -n "$MANGA_ID" ] && [ -n "$COMMENT_ID" ]; then
    # Delete comment (cleanup)
    echo -e "\n--- Cleanup: Delete Comment ---"
    api_test "DELETE" "$API_V1/manga/$MANGA_ID/comments/$COMMENT_ID" "200" "" \
        "Authorization: Bearer $AUTH_TOKEN" "Delete Comment" > /dev/null
    
    # Delete manga (cleanup)
    echo -e "\n--- Cleanup: Delete Manga ---"
    api_test "DELETE" "$API_V1/manga/$MANGA_ID" "200" "" \
        "Authorization: Bearer $AUTH_TOKEN" "Delete Manga" > /dev/null
fi

echo -e "\n${BLUE}📊 Test Results Summary${NC}"
echo "=================================================="
echo -e "Total Tests: ${BLUE}$TOTAL_TESTS${NC}"
echo -e "Passed: ${GREEN}$PASSED_TESTS${NC}"
echo -e "Failed: ${RED}$FAILED_TESTS${NC}"
echo -e "Success Rate: ${BLUE}$(( PASSED_TESTS * 100 / TOTAL_TESTS ))%${NC}"

if [ $FAILED_TESTS -eq 0 ]; then
    echo -e "\n${GREEN}🎉 All tests passed! API is working correctly.${NC}"
    exit 0
else
    echo -e "\n${RED}❌ Some tests failed. Please check the issues above.${NC}"
    exit 1
fi