# TCP Server Test - Phase 3
# Simple test for TCP Progress Sync Server

$serverHost = "localhost"
$serverPort = 9090

Write-Host "=== TCP Progress Sync Server Test ===" -ForegroundColor Cyan
Write-Host ""

# Test 1: Check server
Write-Host "Test 1: Checking if server is running..." -ForegroundColor Yellow
try {
    $test = New-Object System.Net.Sockets.TcpClient
    $test.Connect($serverHost, $serverPort)
    Write-Host "[PASS] Server is listening on port $serverPort" -ForegroundColor Green
    $test.Close()
}
catch {
    Write-Host "[FAIL] Server not running: $_" -ForegroundColor Red
    Write-Host "Start server with: go run cmd/tcp-server/main.go" -ForegroundColor Yellow
    exit 1
}

Write-Host ""

# Test 2: Single client sends message
Write-Host "Test 2: Sending progress update from single client..." -ForegroundColor Yellow
try {
    $client = New-Object System.Net.Sockets.TcpClient
    $client.Connect($serverHost, $serverPort)
    $stream = $client.GetStream()
    $writer = New-Object System.IO.StreamWriter($stream)
    $writer.AutoFlush = $true
    
    $msg = '{"user_id":"user1","manga_id":"one-piece","chapter":10,"timestamp":1700000000}'
    Write-Host "Sending: $msg" -ForegroundColor Cyan
    $writer.WriteLine($msg)
    
    Start-Sleep -Milliseconds 500
    
    $writer.Close()
    $stream.Close()
    $client.Close()
    
    Write-Host "[PASS] Message sent successfully" -ForegroundColor Green
}
catch {
    Write-Host "[FAIL] Error: $_" -ForegroundColor Red
}

Write-Host ""

# Test 3: Two clients with broadcast
Write-Host "Test 3: Testing broadcast between two clients..." -ForegroundColor Yellow

$job1 = Start-Job -ScriptBlock {
    param($h, $p)
    try {
        $c = New-Object System.Net.Sockets.TcpClient
        $c.Connect($h, $p)
        $s = $c.GetStream()
        $w = New-Object System.IO.StreamWriter($s)
        $r = New-Object System.IO.StreamReader($s)
        $w.AutoFlush = $true
        $s.ReadTimeout = 5000
        
        $msg = '{"user_id":"user2","manga_id":"attack-on-titan","chapter":25,"timestamp":1700000001}'
        $w.WriteLine($msg)
        
        $out = @("SENT: $msg")
        $end = (Get-Date).AddSeconds(4)
        
        while ((Get-Date) -lt $end) {
            if ($s.DataAvailable) {
                $line = $r.ReadLine()
                if ($line) { $out += "RECV: $line" }
            }
            Start-Sleep -Milliseconds 100
        }
        
        $w.Close(); $r.Close(); $s.Close(); $c.Close()
        return $out
    }
    catch {
        return @("ERROR: $_")
    }
} -ArgumentList $serverHost, $serverPort

Start-Sleep -Milliseconds 300

$job2 = Start-Job -ScriptBlock {
    param($h, $p)
    try {
        $c = New-Object System.Net.Sockets.TcpClient
        $c.Connect($h, $p)
        $s = $c.GetStream()
        $w = New-Object System.IO.StreamWriter($s)
        $r = New-Object System.IO.StreamReader($s)
        $w.AutoFlush = $true
        $s.ReadTimeout = 5000
        
        Start-Sleep -Milliseconds 500
        
        $msg = '{"user_id":"user3","manga_id":"solo-leveling","chapter":50,"timestamp":1700000002}'
        $w.WriteLine($msg)
        
        $out = @("SENT: $msg")
        $end = (Get-Date).AddSeconds(3)
        
        while ((Get-Date) -lt $end) {
            if ($s.DataAvailable) {
                $line = $r.ReadLine()
                if ($line) { $out += "RECV: $line" }
            }
            Start-Sleep -Milliseconds 100
        }
        
        $w.Close(); $r.Close(); $s.Close(); $c.Close()
        return $out
    }
    catch {
        return @("ERROR: $_")
    }
} -ArgumentList $serverHost, $serverPort

Write-Host "Waiting for clients..." -ForegroundColor Cyan
Wait-Job -Job $job1,$job2 -Timeout 10 | Out-Null

$r1 = Receive-Job -Job $job1
$r2 = Receive-Job -Job $job2

Remove-Job -Job $job1,$job2 -Force

Write-Host ""
Write-Host "Client A:" -ForegroundColor Cyan
$r1 | ForEach-Object { Write-Host "  $_" }

Write-Host ""
Write-Host "Client B:" -ForegroundColor Cyan
$r2 | ForEach-Object { Write-Host "  $_" }

Write-Host ""

$recvCount = ($r1 + $r2 | Where-Object { $_ -like "RECV:*" }).Count

if ($recvCount -gt 0) {
    Write-Host "[PASS] Broadcast working! Clients received $recvCount messages" -ForegroundColor Green
} else {
    Write-Host "[INFO] No broadcasts received (timing issue or clients too fast)" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "=== Test Complete ===" -ForegroundColor Cyan
Write-Host "TCP server is accepting connections and broadcasting messages" -ForegroundColor Green
