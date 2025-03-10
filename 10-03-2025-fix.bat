@echo off
:: Check for administrator privileges
net session >nul 2>&1
if %errorLevel% neq 0 (
	echo Please run this script as an administrator.
	exit /b 1
)

del "C:\Program Files\SafeExamBrowser\Application\un-patch-3.9.0.exe"
echo Successfully removed the unnecessary file.