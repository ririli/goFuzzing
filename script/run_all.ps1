$ErrorActionPreference = "Stop"

# 切换到项目根目录（脚本所在目录的上一级）
$root = Split-Path -Parent $PSScriptRoot
Push-Location $root

try {
    $targets = @("cockroach", "etcd", "grpc", "istio", "kubernetes", "moby", "serving")
    $resultDir = "$root/zgortResult"

    New-Item -ItemType Directory -Force -Path $resultDir | Out-Null

    foreach ($name in $targets) {
        Write-Host "============================================"
        Write-Host "=== [$name] bins ==="
        Write-Host "============================================"
        ./bin/fuzz --task bins --path "testdata/gobench/nonblocking/$name" -o "testbins/$name"

        Write-Host "============================================"
        Write-Host "=== [$name] full ==="
        Write-Host "============================================"
        ./bin/fuzz --task full --path "testbins/$name" > "$resultDir/$name.txt"

        Write-Host "=== [$name] done ==="
        Write-Host ""
    }

    Write-Host "All done. Results saved to $resultDir/"
} finally {
    Pop-Location
}
