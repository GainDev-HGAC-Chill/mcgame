@echo off
setlocal
pushd "%~dp0"
if errorlevel 1 exit /b 1
set "PATH=%~dp0bin;%ProgramFiles%\Docker\Docker\resources\bin;%LOCALAPPDATA%\Programs\DockerDesktop\resources\bin;%PATH%"

where go >nul 2>nul
if errorlevel 1 (
    echo Go was not found on PATH. Install the Go version required by go.mod.
    pause
    popd
    exit /b 1
)
echo Updating launcher...
go build -o bin/game.exe ./cmd/game
if errorlevel 1 (
    echo Launcher build failed.
    pause
    popd
    exit /b 1
)

if /i "%~1"=="status" (
    bin\game.exe status
    goto finish
)
if /i "%~1"=="check" (
    bin\game.exe check
    goto finish
)

:menu
cls
echo ================= MCGAME =================
echo 1. Start server - clean build and start
echo 2. Stop server - backup database first
echo 3. Reset server - backup, stop, build and start
echo 4. Open Flash Client - 30 FPS
echo 5. Restore database
echo 6. Select connection - Local / LAN / Radmin / Online VPS
echo 0. Exit
echo.
echo Database backups: .local\backups\database
echo Git snapshot: supabase\backups\mcgame-live.dump
echo Docker Desktop must be running for server and database actions.
echo.
choice /c 1234560 /n /m "Select: "
if errorlevel 7 goto quit
if errorlevel 6 goto connection
if errorlevel 5 goto restore
if errorlevel 4 goto client30
if errorlevel 3 goto reset
if errorlevel 2 goto stop
if errorlevel 1 goto start
goto quit

:start
bin\game.exe start
goto result

:stop
bin\game.exe stop
goto result

:reset
bin\game.exe reset
goto result

:client30
bin\game.exe client30
goto result

:restore
echo Restore creates a safety backup before replacing the current database.
bin\game.exe restore
goto result

:connection
cls
echo ============== CONNECTION ==============
echo 1. Local
echo 2. LAN
echo 3. Radmin VPN
echo 4. Online VPS
echo 0. Back
echo.
choice /c 12340 /n /m "Select: "
if errorlevel 5 goto menu
if errorlevel 4 goto online
if errorlevel 3 goto radmin
if errorlevel 2 goto lan
if errorlevel 1 goto local
goto menu

:local
bin\game.exe local
goto result

:lan
echo Enter the server private Ethernet IPv4, not a VPN address.
set "MCGAME_LAN_IP="
set /p "MCGAME_LAN_IP=LAN IPv4: "
bin\game.exe lan
goto result

:radmin
echo Both players must join the same Radmin VPN network.
echo The launcher auto-detects the active Radmin IPv4.
set "MCGAME_RADMIN_IP="
bin\game.exe radmin
goto result

:online
echo WARNING: This exposes game HTTP and RTMPE ports publicly.
echo TLS, firewall, domain proxy and security hardening are not automatic.
set "MCGAME_ONLINE_HOST="
set /p "MCGAME_ONLINE_HOST=Public IPv4 or DNS hostname: "
bin\game.exe online
goto result

:result
if errorlevel 1 echo Operation failed. Review the message above.
echo.
echo Connection profile changes apply on Start or Reset.
pause
goto menu

:finish
set "RESULT=%ERRORLEVEL%"
popd
exit /b %RESULT%

:quit
popd
exit /b 0
