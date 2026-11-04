@echo off
setlocal
cd /d "%~dp0"
where node.exe >nul 2>nul
if errorlevel 1 (
  echo Install Node.js 22.12 or newer from https://nodejs.org first.
  echo Then run this file again.
  pause
  exit /b 1
)
node -e "const v=process.versions.node.split('.').map(Number); process.exit(v[0] < 22 || (v[0] === 22 && v[1] < 12) ? 1 : 0)"
if errorlevel 1 (
  echo Update Node.js to version 22.12 or newer from https://nodejs.org.
  pause
  exit /b 1
)
if not exist "node_modules\.bin\vite.cmd" (
  echo Preparing the frontend. The first run needs an internet connection.
  call npm.cmd ci
  if errorlevel 1 (
    echo Installation did not finish. Check the error above.
    pause
    exit /b 1
  )
)
echo Opening QueueLite UI preview. Keep this window open while reviewing.
call npm.cmd run preview:ui -- --open
if errorlevel 1 (
  echo Preview stopped with an error. Check the message above.
  pause
)
