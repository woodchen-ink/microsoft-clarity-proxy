# PowerShell 脚本 - 测试 IP 转发

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "IP 转发测试" -ForegroundColor Cyan
Write-Host "========================================`n" -ForegroundColor Cyan

# 测试 1: 不带任何 IP 头
Write-Host "1. 测试不带 IP 头的请求..." -ForegroundColor Yellow
try {
    $response = Invoke-WebRequest -Uri "http://127.0.0.1:8081/health" -UseBasicParsing
    Write-Host "   ✅ 响应: $($response.Content)" -ForegroundColor Green
} catch {
    Write-Host "   ❌ 失败: $($_.Exception.Message)" -ForegroundColor Red
}

# 测试 2: 模拟 nginx 传递的 X-Real-IP
Write-Host "`n2. 测试带 X-Real-IP 的请求..." -ForegroundColor Yellow
Write-Host "   模拟客户端 IP: 1.2.3.4" -ForegroundColor Gray
try {
    $headers = @{
        "X-Real-IP" = "1.2.3.4"
        "Origin" = "https://woodchen.ink"
    }
    $response = Invoke-WebRequest -Uri "http://127.0.0.1:8081/ms/p" `
        -Method POST `
        -Headers $headers `
        -UseBasicParsing
    Write-Host "   ✅ 请求成功" -ForegroundColor Green
    Write-Host "   查看 Go 代理日志,应该看到:" -ForegroundColor Gray
    Write-Host "   [IP Debug] X-Real-IP: 1.2.3.4" -ForegroundColor Gray
} catch {
    Write-Host "   ⚠️  请求失败 (这是正常的,Clarity 可能拒绝空请求)" -ForegroundColor Yellow
}

# 测试 3: 模拟 CDN 传递的 X-Forwarded-For
Write-Host "`n3. 测试带 X-Forwarded-For 的请求..." -ForegroundColor Yellow
Write-Host "   模拟 IP 链: 5.6.7.8 -> CDN -> nginx" -ForegroundColor Gray
try {
    $headers = @{
        "X-Forwarded-For" = "5.6.7.8"
        "X-Real-IP" = "5.6.7.8"
        "Origin" = "https://woodchen.ink"
    }
    $response = Invoke-WebRequest -Uri "http://127.0.0.1:8081/ms/p" `
        -Method POST `
        -Headers $headers `
        -UseBasicParsing
    Write-Host "   ✅ 请求成功" -ForegroundColor Green
    Write-Host "   查看 Go 代理日志,应该看到:" -ForegroundColor Gray
    Write-Host "   [IP Debug] X-Forwarded-For: 5.6.7.8" -ForegroundColor Gray
} catch {
    Write-Host "   ⚠️  请求失败 (这是正常的,Clarity 可能拒绝空请求)" -ForegroundColor Yellow
}

Write-Host "`n========================================" -ForegroundColor Cyan
Write-Host "测试完成" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan

Write-Host "`n下一步:" -ForegroundColor Yellow
Write-Host "1. 重启 Go 代理服务" -ForegroundColor Gray
Write-Host "2. 查看 Go 代理的日志输出" -ForegroundColor Gray
Write-Host "3. 访问你的网站 https://woodchen.ink" -ForegroundColor Gray
Write-Host "4. 观察日志中的 [IP Debug] 信息" -ForegroundColor Gray
Write-Host "5. 确认 iputil.GetClientIP 返回的是真实客户端 IP" -ForegroundColor Gray
Write-Host "`n6. 如果还是显示服务器 IP,检查:" -ForegroundColor Gray
Write-Host "   - nginx 是否配置了 set_real_ip_from" -ForegroundColor Gray
Write-Host "   - nginx 是否配置了 real_ip_header X-Forwarded-For" -ForegroundColor Gray
Write-Host "   - EdgeOne CDN 是否正确传递了 X-Forwarded-For" -ForegroundColor Gray
