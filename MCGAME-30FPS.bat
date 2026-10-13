@echo off
setlocal
pushd "%~dp0"
if errorlevel 1 exit /b 1
echo Experimental 30 FPS client - original client remains unchanged.
echo Close other Flash windows first. Revert with MCGAME.bat option 2.
if not exist "bin\game.exe" (
    echo Missing bin\game.exe. Build with go build -o bin/game.exe ./cmd/game
    pause
    popd
    exit /b 1
)
bin\game.exe client30
set "RESULT=%ERRORLEVEL%"
if not "%RESULT%"=="0" pause
popd
exit /b %RESULT%
