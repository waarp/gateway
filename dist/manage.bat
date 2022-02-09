@echo off

SET CURDIR=%~dp0
SET DAEMONNAME=waarp-gatewayd.exe
SET DAEMONPATH=%CURDIR%\..\bin\%DAEMONNAME%
SET DAEMON_PARAMS=server -c "%CURDIR%\..\etc\gatewayd.ini"

cd %CURDIR%\..

SET "PATH=%CURDIR%;%CURDIR%\..\share;%PATH%"

%DAEMONPATH% %DAEMON_PARAMS%
