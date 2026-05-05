@echo off
setlocal enabledelayedexpansion
chcp 65001 >nul 2>&1

REM ============================================================
REM  FrontLeaves Sync — 自升级脚本 (Windows)
REM  这是玩家唯一需要下载的文件，放在与 .minecraft/ 同级目录下运行。
REM ============================================================

set "SERVER_BASE=https://game.frontleaves.com/api/v1"
set "METADATA_URL=%SERVER_BASE%/sync/scripts/metadata"
set "DOWNLOAD_URL=%SERVER_BASE%/sync/download"
set "BINARY_NAME=frontleaves-sync.exe"

REM -------------------- 检测平台 --------------------
set "OS=windows"
if "%PROCESSOR_ARCHITECTURE%"=="AMD64" (
    set "ARCH=amd64"
) else if "%PROCESSOR_ARCHITECTURE%"=="ARM64" (
    set "ARCH=arm64"
) else (
    set "ARCH=amd64"
)
set "SUFFIX=%OS%-%ARCH%"
set "TARGET_NAME=frontleaves-sync-%SUFFIX%.exe"

echo.
echo   FrontLeaves Sync 自升级工具
echo.

REM -------------------- 确定路径（放到 .minecraft 同级的 update\ 目录） --------------------
set "SCRIPT_DIR=%~dp0"
set "SCRIPT_DIR=%SCRIPT_DIR:~0,-1%"

set "MC_PARENT="
if exist "%SCRIPT_DIR%\.minecraft" (
    set "MC_PARENT=%SCRIPT_DIR%"
) else if exist "%SCRIPT_DIR%\..\.minecraft" (
    for %%i in ("%SCRIPT_DIR%\..") do set "MC_PARENT=%%~fi"
) else if exist "%SCRIPT_DIR%\..\..\.minecraft" (
    for %%i in ("%SCRIPT_DIR%\..\..") do set "MC_PARENT=%%~fi"
)

if not defined MC_PARENT (
    echo [ERROR] 找不到 .minecraft/ 目录。
    echo 请将本脚本放在 MC 游戏目录下运行（与 .minecraft/ 同级或其子目录内）。
    goto :fail
)

set "UPDATE_DIR=!MC_PARENT!\update"
if not exist "!UPDATE_DIR!" mkdir "!UPDATE_DIR!"
set "BINARY_PATH=!UPDATE_DIR!\%BINARY_NAME%"

REM -------------------- 检查 curl --------------------
where curl >nul 2>&1
if errorlevel 1 (
    echo [ERROR] 需要 curl 命令，Windows 10+ 自带，请检查系统版本。
    goto :fail
)

REM -------------------- 获取远程文件信息 --------------------
echo [INFO] 正在获取服务端文件列表...

set "TEMP_JSON=%TEMP%\frontleaves-sync-metadata.json"
curl -fsSL -o "%TEMP_JSON%" "%METADATA_URL%" 2>nul
if errorlevel 1 (
    echo [ERROR] 无法连接服务端，请检查网络连接。
    goto :fail
)

REM 从 JSON 查找匹配 TARGET_NAME 的文件
set "REMOTE_PATH="
set "REMOTE_HASH="
call :parse_json "%TEMP_JSON%" "%TARGET_NAME%"

del /f /q "%TEMP_JSON%" 2>nul

if not defined REMOTE_PATH (
    echo [ERROR] 服务端未找到 %TARGET_NAME%，该平台可能尚未发布。
    goto :fail
)

echo [INFO] 平台: %SUFFIX%

REM -------------------- 判断是否需要下载 --------------------
set "NEED_DOWNLOAD=0"

if not exist "%BINARY_PATH%" (
    echo [INFO] 首次运行，正在下载同步器...
    set "NEED_DOWNLOAD=1"
) else (
    REM 哈希比对（需要 certutil）
    if defined REMOTE_HASH (
        set "EXPECTED_HEX=!REMOTE_HASH:sha256:=!"
        for /f "skip=1 delims=" %%h in ('certutil -hashfile "%BINARY_PATH%" SHA256 2^>nul') do (
            set "ACTUAL_HASH=%%h"
            set "ACTUAL_HASH=!ACTUAL_HASH: =!"
            goto :hash_compare
        )
        :hash_compare
        if defined ACTUAL_HASH (
            if /i not "!ACTUAL_HASH!"=="!EXPECTED_HEX!" (
                echo [INFO] 发现新版本，正在更新...
                set "NEED_DOWNLOAD=1"
            ) else (
                echo [OK] 已是最新版本，无需更新。
            )
        ) else (
            REM 无法计算哈希，跳过比对直接下载
            set "NEED_DOWNLOAD=1"
        )
    ) else (
        set "NEED_DOWNLOAD=1"
    )
)

if "%NEED_DOWNLOAD%"=="0" goto :launch

REM -------------------- 下载 --------------------
echo.

REM URL 编码路径
set "ENCODED_PATH=!REMOTE_PATH!"
where powershell >nul 2>&1
if not errorlevel 1 (
    for /f "usebackq delims=" %%e in (`powershell -NoProfile -Command "[uri]::EscapeDataString('!REMOTE_PATH!')"`) do set "ENCODED_PATH=%%e"
)
set "DL_URL=%DOWNLOAD_URL%?path=!ENCODED_PATH!"

echo [INFO] 正在下载 %TARGET_NAME%...
echo [INFO] 下载地址: %DL_URL%

set "TEMP_FILE=%TEMP%\frontleaves-sync-update.exe"
curl -fSL --progress-bar -o "%TEMP_FILE%" "%DL_URL%"
if errorlevel 1 (
    echo [ERROR] 下载失败，请检查网络连接。
    del /f /q "%TEMP_FILE%" 2>nul
    goto :fail
)

REM 校验文件大小
for %%f in ("%TEMP_FILE%") do set "FILE_SIZE=%%~zf"
if "%FILE_SIZE%"=="0" (
    echo [ERROR] 下载的文件为空。
    del /f /q "%TEMP_FILE%" 2>nul
    goto :fail
)

REM 哈希校验
if defined REMOTE_HASH (
    for /f "skip=1 delims=" %%h in ('certutil -hashfile "%TEMP_FILE%" SHA256 2^>nul') do (
        set "ACTUAL_HASH=%%h"
        set "ACTUAL_HASH=!ACTUAL_HASH: =!"
        goto :download_hash_check
    )
    :download_hash_check
    if defined ACTUAL_HASH (
        set "EXPECTED_HEX=!REMOTE_HASH:sha256:=!"
        if /i not "!ACTUAL_HASH!"=="!EXPECTED_HEX!" (
            echo [ERROR] 哈希校验失败！文件可能已损坏。
            del /f /q "%TEMP_FILE%" 2>nul
            goto :fail
        )
        echo [OK] 哈希校验通过
    )
)

REM 替换
if exist "%BINARY_PATH%" (
    set "BACKUP=%BINARY_PATH%.bak"
    copy /y "%BINARY_PATH%" "!BACKUP!" >nul 2>&1
    move /y "%TEMP_FILE%" "%BINARY_PATH%" >nul 2>&1
    if errorlevel 1 (
        echo [ERROR] 替换文件失败，可能被占用。
        move /y "!BACKUP!" "%BINARY_PATH%" >nul 2>&1
        del /f /q "%TEMP_FILE%" 2>nul
        echo [INFO] 请关闭所有 frontleaves-sync 窗口后重试。
        goto :fail
    )
    del /f /q "!BACKUP!" 2>nul
) else (
    move /y "%TEMP_FILE%" "%BINARY_PATH%" >nul 2>&1
    if errorlevel 1 (
        del /f /q "%TEMP_FILE%" 2>nul
        echo [ERROR] 写入文件失败，请以管理员身份运行。
        goto :fail
    )
)

echo [OK] 下载完成: %BINARY_NAME%

REM -------------------- 启动同步器 --------------------
:launch
echo.
echo [OK] 启动同步器...
echo.

REM 切换到 .minecraft 父目录，确保二进制能找到正确的路径
cd /d "!MC_PARENT!"

REM 如果二进制不在 update/ 目录中（直接运行 .bat），则从 update/ 启动
if exist "!UPDATE_DIR!\%BINARY_NAME%" (
    "!UPDATE_DIR!\%BINARY_NAME%"
) else (
    echo [ERROR] 找不到同步器: !UPDATE_DIR!\%BINARY_NAME%
    goto :fail
)

:done
endlocal
pause
exit /b 0

:fail
del /f /q "%TEMP_JSON%" 2>nul
endlocal
pause
exit /b 1

REM -------------------- JSON 解析子程序（双重兼容） --------------------
:parse_json
set "PJ_FILE=%~1"
set "PJ_TARGET=%~2"

REM 方法1: PowerShell（Windows 8+ 内置，推荐）
where powershell >nul 2>&1
if not errorlevel 1 (
    for /f "usebackq tokens=1,2 delims=|" %%p in (`powershell -NoProfile -Command "try{$j=Get-Content -Raw '!PJ_FILE!'|ConvertFrom-Json;$f=$j.data.files|Where-Object{$_.name -eq '!PJ_TARGET!'}|Select-Object -First 1;if($f){Write-Output ($f.path+'|'+$f.hash)}}catch{}"`) do (
        set "REMOTE_PATH=%%p"
        set "REMOTE_HASH=%%q"
    )
    if defined REMOTE_PATH goto :eof
)

REM 方法2: JScript via cscript（Windows XP+ 均可用，无外部依赖）
set "PJ_JS=%TEMP%\fls-parse.js"
echo var fso=new ActiveXObject("Scripting.FileSystemObject");>"!PJ_JS!"
echo var s=fso.OpenTextFile(WScript.Arguments(0),1).ReadAll();>>"!PJ_JS!"
echo var t=WScript.Arguments(1);>>"!PJ_JS!"
echo var n='"name":"'+t+'"';>>"!PJ_JS!"
echo var i=s.indexOf(n);>>"!PJ_JS!"
echo if(i^>-1){>>"!PJ_JS!"
echo   var a=s.lastIndexOf('{',i);>>"!PJ_JS!"
echo   var b=s.indexOf('}',i);>>"!PJ_JS!"
echo   var o=s.substring(a,b+1);>>"!PJ_JS!"
echo   var p=o.match(/"path":"(.+?)"/);>>"!PJ_JS!"
echo   var h=o.match(/"hash":"(.+?)"/);>>"!PJ_JS!"
echo   if(p^&^&h)WScript.Echo(p[1]+'^|'+h[1]);>>"!PJ_JS!"
echo }>>"!PJ_JS!"
for /f "usebackq tokens=1,2 delims=|" %%p in (`cscript //nologo //E:JScript "!PJ_JS!" "!PJ_FILE!" "!PJ_TARGET!" 2^>nul`) do (
    set "REMOTE_PATH=%%p"
    set "REMOTE_HASH=%%q"
)
del /f /q "!PJ_JS!" 2>nul
goto :eof
