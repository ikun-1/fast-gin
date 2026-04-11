@echo off
setlocal EnableExtensions EnableDelayedExpansion

set "CONFIG=settings-dev.yaml"
if not "%~1"=="" set "CONFIG=%~1"

if not exist "%CONFIG%" (
  echo [run.bat] Config file not found: %CONFIG%
  exit /b 1
)

set "SWAGGER_ENABLED=false"
set "IN_SYSTEM=0"
set "IN_SWAGGER=0"

for /f "usebackq delims=" %%L in ("%CONFIG%") do (
  set "LINE=%%L"

  if "!LINE:~0,7!"=="system:" (
    set "IN_SYSTEM=1"
    set "IN_SWAGGER=0"
  ) else if !IN_SYSTEM! EQU 1 (
    if "!LINE:~0,2!"=="  " (
      if "!LINE:~0,10!"=="  swagger:" (
        set "IN_SWAGGER=1"
      ) else if !IN_SWAGGER! EQU 1 (
        if "!LINE:~0,12!"=="    enabled:" (
          for /f "tokens=2 delims=:" %%V in ("!LINE!") do (
            set "VAL=%%V"
            set "VAL=!VAL: =!"
            if /I "!VAL!"=="true" set "SWAGGER_ENABLED=true"
          )
        )
      )
    ) else (
      set "IN_SYSTEM=0"
      set "IN_SWAGGER=0"
    )
  )
)

if /I "%SWAGGER_ENABLED%"=="true" (
  echo [run.bat] swagger.enabled=true, run swag init...
  swag init
  if errorlevel 1 (
    echo [run.bat] swag init failed
    exit /b 1
  )
) else (
  echo [run.bat] swagger.enabled=false, skip swag init
)

echo [run.bat] Start server...
go run .\main.go -f "%CONFIG%"
exit /b %ERRORLEVEL%
