@echo off
setlocal
pushd "%~dp0"
if errorlevel 1 exit /b 1
set "PATH=%~dp0bin;%ProgramFiles%\Docker\Docker\resources\bin;%LOCALAPPDATA%\Programs\DockerDesktop\resources\bin;%PATH%"
if not exist "bin\game.exe" (
    echo Missing bin\game.exe. Build with:
    echo go build -o bin/game.exe ./cmd/game
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
echo 1. Start server
echo 2. Open Flash Client
echo F. Open Flash Client - 30 FPS experimental
echo 3. Stop server - keep data
echo 4. Restart server - keep data
echo 5. Status / LAN / Radmin IP / Client URL
echo 6. Logs
echo 7. Check Docker / Database / Redis / Nginx
echo 8. Select Local profile
echo 9. Select LAN profile
echo R. Select Radmin VPN profile
echo 0. Exit
echo.
echo Docker Desktop must be running.
echo If WSL2 was just enabled, restart Windows yourself first.
echo No database reset or volume deletion is provided here.
echo.
choice /c 1234567890RF /n /m "Select: "
if errorlevel 12 goto client30
if errorlevel 11 goto radmin
if errorlevel 10 goto quit
if errorlevel 9 goto lan
if errorlevel 8 goto local
if errorlevel 7 goto check
if errorlevel 6 goto logs
if errorlevel 5 goto status
if errorlevel 4 goto restart
if errorlevel 3 goto stop
if errorlevel 2 goto client
if errorlevel 1 goto start
goto quit

:start
bin\game.exe start
goto result
:client
bin\game.exe client
goto result
:client30
bin\game.exe client30
goto result
:stop
bin\game.exe stop
goto result
:restart
bin\game.exe restart
goto result
:status
bin\game.exe status
goto result
:logs
bin\game.exe logs
goto result
:check
bin\game.exe check
goto result
:local
bin\game.exe local
goto result
:lan
echo Verify Local login/gameplay before enabling LAN.
echo Enter the server's private Ethernet IPv4, not a VPN address.
set "MCGAME_LAN_IP="
set /p "MCGAME_LAN_IP=LAN IPv4: "
bin\game.exe lan
goto result

:radmin
echo Both players must join the same Radmin VPN network.
echo Auto-detects the server's active Radmin IPv4.
set "MCGAME_RADMIN_IP="
bin\game.exe radmin
goto result

:result
if errorlevel 1 echo Operation failed. Review the message above.
echo.
echo Profile changes require Start to apply, not Restart.
pause
goto menu

:finish
set "RESULT=%ERRORLEVEL%"
popd
exit /b %RESULT%

:quit
popd
exit /b 0
