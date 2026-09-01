# 并行启动两个项目的 run_full（后台长跑任务，无需等待结束）
$env:Path = 'D:\Program Files\GO\go1.25.1\bin;' + $env:Path
Set-Location d:\gopath\src\gopie

Start-Process powershell -ArgumentList @(
    '-NoProfile','-ExecutionPolicy','Bypass','-File','d:\gopath\src\gopie\scripts\run_full.ps1',
    '-BinDir','testbins\beegoF',
    '-OutDir','D:\gopath\src\real-projects\BEEGO\beegoF\gopieRes',
    '-Granularity','function',
    '-Timeout','300',
    '-FuzzTime','1800'
) -RedirectStandardOutput d:\gopath\src\gopie\runF.log -RedirectStandardError d:\gopath\src\gopie\runF.err.log
Write-Host "run_full(F) launched"

Start-Process powershell -ArgumentList @(
    '-NoProfile','-ExecutionPolicy','Bypass','-File','d:\gopath\src\gopie\scripts\run_full.ps1',
    '-BinDir','testbins\beegoG',
    '-OutDir','D:\gopath\src\real-projects\BEEGO\beegoG\gopieRes',
    '-Granularity','goroutine',
    '-Timeout','300',
    '-FuzzTime','1800'
) -RedirectStandardOutput d:\gopath\src\gopie\runG.log -RedirectStandardError d:\gopath\src\gopie\runG.err.log
Write-Host "run_full(G) launched"
