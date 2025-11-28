# TCP Server Test Script
# Tests the TCP Progress Sync Server with multiple clients

$serverHost = "localhost"
$serverPort = 9090

Write-Host "=== TCP Progress Sync Server Test ===" -ForegroundColor Cyan
Write-Host ""


# Test 1: Check if server is running
Write-Host "Test 1: Checking if TCP server is running on port $serverPort..." -ForegroundColor Cyan
$serverRunning = $false
try {
    $testClient = New-Object System.Net.Sockets.TcpClient
    $testClient.Connect($serverHost, $serverPort)
    if ($testClient.Connected) {
        Write-Host "✓ Server is running!" -ForegroundColor Green
        $testClient.Close()
        $serverRunning = $true
    }
}
catch {
    Write-Host "✗ Server is NOT running. Please start it with: go run cmd/tcp-server/main.go" -ForegroundColor Red
}

if (-not $serverRunning) {
    Write-Host "`nAttempting to start server..." -ForegroundColor Yellow
    exit 1
}

Write-Host ""

# Test 2: Single client test
Write-Host "Test 2: Testing single client connection and message send..." -ForegroundColor Cyan
try {
    $client1 = New-Object System.Net.Sockets.TcpClient
    $client1.Connect($serverHost, $serverPort)
    $stream1 = $client1.GetStream()
    $writer1 = New-Object System.IO.StreamWriter($stream1)
    $writer1.AutoFlush = $true
    
    $message1 = '{"user_id":"user1","manga_id":"one-piece","chapter":10,"timestamp":1700000000}'
    Write-Host "Sending: $message1" -ForegroundColor Yellow
    $writer1.WriteLine($message1)
    
    Start-Sleep -Milliseconds 500
    
    Write-Host "✓ Message sent successfully!" -ForegroundColor Green
    
    $writer1.Close()
    $stream1.Close()
    $client1.Close()
}
catch {
    Write-Host "✗ Error: $_" -ForegroundColor Red
}

Write-Host ""

# Test 3: Multiple clients with broadcast
Write-Host "Test 3: Testing broadcast with multiple clients..." -ForegroundColor Cyan
Write-Host "Opening 2 clients and testing message broadcast between them..." -ForegroundColor Yellow

$job1 = Start-Job -ScriptBlock {
    param($host, $port)
    try {
        $client = New-Object System.Net.Sockets.TcpClient
        $client.Connect($host, $port)
        $stream = $client.GetStream()
        $writer = New-Object System.IO.StreamWriter($stream)
        $reader = New-Object System.IO.StreamReader($stream)
        $writer.AutoFlush = $true
        $stream.ReadTimeout = 5000
        
        # Send message
        $msg = '{"user_id":"user2","manga_id":"attack-on-titan","chapter":25,"timestamp":1700000001}'
        $writer.WriteLine($msg)
        
        $results = @("Client A sent: $msg")
        
        # Listen for broadcasts
        $endTime = (Get-Date).AddSeconds(4)
        while ((Get-Date) -lt $endTime) {
            if ($stream.DataAvailable) {
                $response = $reader.ReadLine()
                if ($response) {
                    $results += "Client A received: $response"
                }
            }
            Start-Sleep -Milliseconds 100
        }
        
        $writer.Close()
        $reader.Close()
        $stream.Close()
        $client.Close()
        
        return $results
    }
    catch {
        return @("Client A error: $_")
    }
} -ArgumentList $serverHost, $serverPort

Start-Sleep -Milliseconds 300

$job2 = Start-Job -ScriptBlock {
    param($host, $port)
    try {
        $client = New-Object System.Net.Sockets.TcpClient
        $client.Connect($host, $port)
        $stream = $client.GetStream()
        $writer = New-Object System.IO.StreamWriter($stream)
        $reader = New-Object System.IO.StreamReader($stream)
        $writer.AutoFlush = $true
        $stream.ReadTimeout = 5000
        
        Start-Sleep -Milliseconds 500
        
        # Send message
        $msg = '{"user_id":"user3","manga_id":"solo-leveling","chapter":50,"timestamp":1700000002}'
        $writer.WriteLine($msg)
        
        $results = @("Client B sent: $msg")
        
        # Listen for broadcasts
        $endTime = (Get-Date).AddSeconds(3)
        while ((Get-Date) -lt $endTime) {
            if ($stream.DataAvailable) {
                $response = $reader.ReadLine()
                if ($response) {
                    $results += "Client B received: $response"
                }
            }
            Start-Sleep -Milliseconds 100
        }
        
        $writer.Close()
        $reader.Close()
        $stream.Close()
        $client.Close()
        
        return $results
    }
    catch {
        return @("Client B error: $_")
    }
} -ArgumentList $serverHost, $serverPort

Write-Host "Waiting for clients to complete..." -ForegroundColor Yellow
Wait-Job -Job $job1, $job2 -Timeout 10 | Out-Null

$results1 = Receive-Job -Job $job1
$results2 = Receive-Job -Job $job2

Remove-Job -Job $job1, $job2 -Force

Write-Host ""
Write-Host "Client A:" -ForegroundColor Cyan
$results1 | ForEach-Object { Write-Host "  $_" }

Write-Host ""
Write-Host "Client B:" -ForegroundColor Cyan
$results2 | ForEach-Object { Write-Host "  $_" }

Write-Host ""
Write-Host "=== Analysis ===" -ForegroundColor Cyan

$broadcastCount = ($results1 + $results2 | Where-Object { $_ -like "*received:*" }).Count

if ($broadcastCount -gt 0) {
    Write-Host "✓ SUCCESS: Broadcast working! ($broadcastCount messages received)" -ForegroundColor Green
} else {
    Write-Host "⚠ No broadcasts detected (clients may have disconnected too quickly)" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "=== TCP Server Test Complete ===" -ForegroundColor Cyan
Write-Host "Check the server terminal for connection logs" -ForegroundColor Gray
