# gRPC Service Test Script
# Tests the gRPC MangaService with 3 RPC methods

$grpcServer = "localhost:9092"

Write-Host "=== gRPC Service Test ===" -ForegroundColor Cyan
Write-Host ""
Write-Host "Prerequisites:" -ForegroundColor Yellow
Write-Host "  1. gRPC server must be running: go run cmd/grpc-server/main.go" -ForegroundColor Gray
Write-Host "  2. grpcurl must be installed: go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest" -ForegroundColor Gray
Write-Host ""

# Check if grpcurl is available
try {
    $grpcurlPath = (Get-Command grpcurl -ErrorAction Stop).Source
    Write-Host "[OK] grpcurl found at: $grpcurlPath" -ForegroundColor Green
}
catch {
    Write-Host "[ERROR] grpcurl not found. Please install it:" -ForegroundColor Red
    Write-Host "  go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest" -ForegroundColor Yellow
    Write-Host "  Then restart your terminal to refresh PATH" -ForegroundColor Yellow
    exit 1
}

Write-Host ""

# Test 1: Check if server is running
Write-Host "Test 1: Checking gRPC server..." -ForegroundColor Yellow
try {
    $result = & grpcurl -plaintext $grpcServer list 2>&1 | Out-String
    if ($result -like "*mangahub.v1.MangaService*") {
        Write-Host "[PASS] gRPC server is running and MangaService is available" -ForegroundColor Green
    } else {
        Write-Host "[FAIL] gRPC server not responding properly" -ForegroundColor Red
        Write-Host "Output: $result" -ForegroundColor Gray
        exit 1
    }
}
catch {
    Write-Host "[FAIL] Cannot connect to gRPC server" -ForegroundColor Red
    Write-Host "Make sure server is running: go run cmd/grpc-server/main.go" -ForegroundColor Yellow
    exit 1
}

Write-Host ""

# Test 2: GetManga
Write-Host "Test 2: Testing GetManga RPC..." -ForegroundColor Yellow
try {
    # Using actual UUID from database for One Piece
    $getMangaResp = & grpcurl -plaintext -d '{\"manga_id\":\"3051a7b2-b47f-4e37-9204-231ce56b7dfb\"}' $grpcServer mangahub.v1.MangaService/GetManga 2>&1 | Out-String

    if ($getMangaResp -like '*"title"*') {
        Write-Host "[PASS] GetManga RPC working" -ForegroundColor Green
        # Extract title from response
        if ($getMangaResp -match '"title":\s*"([^"]+)"') {
            Write-Host "  Found manga: $($matches[1])" -ForegroundColor Gray
        }
    } else {
        Write-Host "[FAIL] GetManga did not return expected data" -ForegroundColor Red
        Write-Host "Response: $getMangaResp" -ForegroundColor Gray
    }
}
catch {
    Write-Host "[FAIL] GetManga RPC error: $_" -ForegroundColor Red
}

Write-Host ""

# Test 3: SearchManga
Write-Host "Test 3: Testing SearchManga RPC..." -ForegroundColor Yellow
try {
    $searchResp = & grpcurl -plaintext -d '{\"query\":\"one\",\"limit\":5,\"offset\":0}' $grpcServer mangahub.v1.MangaService/SearchManga 2>&1 | Out-String

    if ($searchResp -like '*"manga"*') {
        Write-Host "[PASS] SearchManga RPC working" -ForegroundColor Green
        # Extract total from response
        if ($searchResp -match '"total":\s*(\d+)') {
            Write-Host "  Total results: $($matches[1])" -ForegroundColor Gray
        }
    } else {
        Write-Host "[FAIL] SearchManga did not return expected data" -ForegroundColor Red
        Write-Host "Response: $searchResp" -ForegroundColor Gray
    }
}
catch {
    Write-Host "[FAIL] SearchManga RPC error: $_" -ForegroundColor Red
}

Write-Host ""

# Test 4: UpdateProgress
Write-Host "Test 4: Testing UpdateProgress RPC..." -ForegroundColor Yellow
try {
    # Using username "reader1" (will be converted to UUID by service)
    $updateJson = '{\"user_id\":\"reader1\",\"manga_id\":\"3051a7b2-b47f-4e37-9204-231ce56b7dfb\",\"current_chapter\":50,\"status\":\"reading\",\"rating\":8}'

    $updateResp = & grpcurl -plaintext -d $updateJson $grpcServer mangahub.v1.MangaService/UpdateProgress 2>&1 | Out-String

    if ($updateResp -like '*"currentChapter"*') {
        Write-Host "[PASS] UpdateProgress RPC working" -ForegroundColor Green
        # Extract chapter from response (camelCase format)
        if ($updateResp -match '"currentChapter":\s*(\d+)') {
            Write-Host "  Updated to chapter: $($matches[1])" -ForegroundColor Gray
        }
    } else {
        Write-Host "[FAIL] UpdateProgress did not return expected data" -ForegroundColor Red
        Write-Host "Response: $updateResp" -ForegroundColor Gray
    }
}
catch {
    Write-Host "[FAIL] UpdateProgress RPC error: $_" -ForegroundColor Red
}

Write-Host ""
Write-Host "=== gRPC Tests Complete ===" -ForegroundColor Cyan
Write-Host "All 3 core gRPC methods have been tested" -ForegroundColor Green
Write-Host ""
Write-Host "Summary:" -ForegroundColor Cyan
Write-Host "  - GetManga: Retrieves single manga by ID" -ForegroundColor Gray
Write-Host "  - SearchManga: Searches with filters and pagination" -ForegroundColor Gray
Write-Host "  - UpdateProgress: Updates user reading progress" -ForegroundColor Gray
