# 并行启动两个 FIBER 项目的 run_full（后台长跑任务，无需等待结束）
$env:Path = 'D:\Program Files\GO\go1.25.1\bin;' + $env:Path
Set-Location d:\gopath\src\gopie

Start-Process powershell -ArgumentList @(
    '-NoProfile','-ExecutionPolicy','Bypass','-File','d:\gopath\src\gopie\scripts\run_full.ps1',
    '-BinDir','testbins\fiberF',
    '-OutDir','D:\gopath\src\real-projects\FIBER\fiberF\gopieRes',
    '-Granularity','function',
    '-Timeout','300',
    '-FuzzTime','1800'
) -RedirectStandardOutput d:\gopath\src\gopie\fiber_runF.log -RedirectStandardError d:\gopath\src\gopie\fiber_runF.err.log
Write-Host "run_full(fiberF) launched"

Start-Process powershell -ArgumentList @(
    '-NoProfile','-ExecutionPolicy','Bypass','-File','d:\gopath\src\gopie\scripts\run_full.ps1',
    '-BinDir','testbins\fiberG',
    '-OutDir','D:\gopath\src\real-projects\FIBER\fiberG\gopieRes',
    '-Granularity','goroutine',
    '-Timeout','300',
    '-FuzzTime','1800'
) -RedirectStandardOutput d:\gopath\src\gopie\fiber_runG.log -RedirectStandardError d:\gopath\src\gopie\fiber_runG.err.log
Write-Host "run_full(fiberG) launched"
