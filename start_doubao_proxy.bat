@echo off
setlocal
cd /d "%~dp0"

REM Doubao websocket configuration
set "DOUBAO_WS_URL=wss://openspeech.bytedance.com/api/v3/sauc/bigmodel_async"
set "DOUBAO_RESOURCE_ID=volc.seedasr.sauc.duration"

REM API Key authentication; AK/SK is not required.
REM Replace the placeholder with your API Key.
set "DOUBAO_API_KEY=your-api-key"

if not exist "proxy.exe" (
    echo [ERROR] proxy.exe not found. Build it first:
    echo go build -o proxy.exe ./cmd/proxy
    exit /b 1
)

echo Starting xaux proxy with Doubao ASR...
proxy.exe --asr_provider=doubao %*
exit /b %ERRORLEVEL%
