@echo off

cd %~dp0

echo [StealthDNS] Initializing...
git submodule update --init --recursive
go mod tidy

echo [StealthDNS] Building OpenNHP SDK from submodule...
call :build_sdk
IF %ERRORLEVEL% NEQ 0 goto :exit

echo [StealthDNS] Building StealthDNS...
:main_build
go build -trimpath -ldflags="-w -s" -v -o release\stealth-dns.exe main.go
IF %ERRORLEVEL% NEQ 0 goto :exit
if not exist release\etc mkdir release\etc
if not exist release\etc\cert mkdir release\etc\cert
if not exist release\sdk mkdir release\sdk
copy  etc\*.* release\etc
copy  sdk\nhp-agent.* release\sdk
copy  etc\cert\rootCA.pem release\etc\cert\
goto :done

:build_sdk
echo [StealthDNS] Building Windows SDK (nhp-agent.dll)...
set CGO_ENABLED=1

cd third_party\opennhp\nhp
go mod tidy
IF %ERRORLEVEL% NEQ 0 exit /b %ERRORLEVEL%

cd ..\endpoints
go mod tidy
IF %ERRORLEVEL% NEQ 0 exit /b %ERRORLEVEL%

go build -a -trimpath -buildmode=c-shared -ldflags="-w -s" -v -o ..\..\..\sdk\nhp-agent.dll agent\main\main.go agent\main\export.go
IF %ERRORLEVEL% NEQ 0 exit /b %ERRORLEVEL%

cd ..\..\..
echo [StealthDNS] Windows SDK built successfully!
exit /b 0

:done
echo [Done] StealthDNS for platform %OS% built!
goto :eof

:exit
echo [Error] Build failed with error code %ERRORLEVEL%
