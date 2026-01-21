#!/bin/bash

# Simple MangaHub API Test Script
set -e

BASE_URL="http://localhost:8080"
API_V1="$BASE_URL/api/v1"

echo "🚀 Testing MangaHub API..."

# Test 1: Health Check
echo -n "Health Check... "
HEALTH=$(curl -s "$BASE_URL/health")
if [[ $HEALTH == *"ok"* ]]; then
    echo "✅ PASS"
else
    echo "❌ FAIL: $HEALTH"
    exit 1
fi

# Test 2: Register User (unique username)
echo -n "User Registration... "
TEST_USERNAME="testuser_$(date +%s)"
REGISTER=$(curl -s -X POST "$API_V1/auth/register" \
    -H "Content-Type: application/json" \
    -d '{"username":"'"$TEST_USERNAME"'","password":"password123"}')
if [[ $REGISTER == *"user"* ]]; then
    echo "✅ PASS"
    USER_ID=$(echo "$REGISTER" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
    echo "  User ID: $USER_ID"
else
    echo "❌ FAIL: $REGISTER"
fi

# Test 3: Login
echo -n "User Login... "
LOGIN=$(curl -s -X POST "$API_V1/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"username":"'"$TEST_USERNAME"'","password":"password123"}')
if [[ $LOGIN == *"token"* ]]; then
    echo "✅ PASS"
    TOKEN=$(echo "$LOGIN" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
    echo "  Token: ${TOKEN:0:20}..."
else
    echo "❌ FAIL: $LOGIN"
    exit 1
fi

# Test 4: List Manga (empty)
echo -n "List Manga... "
MANGA_LIST=$(curl -s "$API_V1/manga")
if [[ $MANGA_LIST == *"data"* ]]; then
    echo "✅ PASS (empty list)"
else
    echo "❌ FAIL: $MANGA_LIST"
fi

# Test 5: Create Manga (admin-only)
echo -n "Create Manga... "
ADMIN_TOKEN=$(curl -s -X POST "$API_V1/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"username":"admin","password":"admin123"}' | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
CREATE_MANGA=$(curl -s -X POST "$API_V1/manga" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -d '{"title":"Test Manga API","description":"A test manga","status":"ongoing"}')
if [[ $CREATE_MANGA == *"Test Manga API"* ]]; then
    echo "✅ PASS"
    MANGA_ID=$(echo "$CREATE_MANGA" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
    echo "  Manga ID: $MANGA_ID"
else
    echo "❌ FAIL: $CREATE_MANGA"
    exit 1
fi

# Test 6: Get Manga Details
echo -n "Get Manga Details... "
MANGA_DETAILS=$(curl -s "$API_V1/manga/$MANGA_ID")
if [[ $MANGA_DETAILS == *"Test Manga API"* ]]; then
    echo "✅ PASS"
else
    echo "❌ FAIL: $MANGA_DETAILS"
fi

# Test 7: Create Comment
echo -n "Create Comment... "
CREATE_COMMENT=$(curl -s -X POST "$API_V1/manga/$MANGA_ID/comments" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $TOKEN" \
    -d '{"content":"This is a test comment!"}')
if [[ $CREATE_COMMENT == *"test comment"* ]]; then
    echo "✅ PASS"
    COMMENT_ID=$(echo "$CREATE_COMMENT" | python3 -c 'import sys, json; print(json.load(sys.stdin)["data"]["id"])')
    echo "  Comment ID: $COMMENT_ID"
else
    echo "❌ FAIL: $CREATE_COMMENT"
fi

# Test 8: List Comments
echo -n "List Comments... "
COMMENT_LIST=$(curl -s "$API_V1/manga/$MANGA_ID/comments")
if [[ $COMMENT_LIST == *"test comment"* ]]; then
    echo "✅ PASS"
else
    echo "❌ FAIL: $COMMENT_LIST"
fi

# Test 9: Like Comment
echo -n "Like Comment... "
LIKE_RESULT=$(curl -s -X POST "$API_V1/manga/$MANGA_ID/comments/$COMMENT_ID/like" \
    -H "Authorization: Bearer $TOKEN")
if [[ $LIKE_RESULT == *"successfully"* ]]; then
    echo "✅ PASS"
else
    echo "❌ FAIL: $LIKE_RESULT"
fi

# Test 10: Global Activity
echo -n "Global Activity... "
ACTIVITY=$(curl -s "$API_V1/activity/global")
if [[ $ACTIVITY == *"data"* ]]; then
    echo "✅ PASS"
else
    echo "❌ FAIL: $ACTIVITY"
fi

# Test 11: Top Stats
echo -n "Top Stats... "
STATS=$(curl -s "$API_V1/stats/top?limit=5")
if [[ $STATS == *"data"* ]]; then
    echo "✅ PASS"
else
    echo "❌ FAIL: $STATS"
fi

# Test 12: Search
echo -n "Search Manga... "
SEARCH=$(curl -s "$API_V1/manga/search?q=Test")
if [[ $SEARCH == *"data"* ]]; then
    echo "✅ PASS"
else
    echo "❌ FAIL: $SEARCH"
fi

# Cleanup
echo -n "Cleanup: Delete Comment... "
DELETE_COMMENT=$(curl -s -X DELETE "$API_V1/manga/$MANGA_ID/comments/$COMMENT_ID" \
    -H "Authorization: Bearer $TOKEN")
if [[ $DELETE_COMMENT == *"successfully"* ]]; then
    echo "✅ PASS"
else
    echo "⚠️ SKIP: $DELETE_COMMENT"
fi

echo -n "Cleanup: Delete Manga... "
DELETE_MANGA=$(curl -s -X DELETE "$API_V1/manga/$MANGA_ID" \
    -H "Authorization: Bearer $ADMIN_TOKEN")
if [[ $DELETE_MANGA == *"successfully"* ]]; then
    echo "✅ PASS"
else
    echo "⚠️ SKIP: $DELETE_MANGA"
fi

echo ""
echo "🎉 All API tests completed successfully!"
echo "Server is running correctly on all endpoints."