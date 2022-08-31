# requires -version 4

#
# Copyright (C) 2018 Ali Abdulkadir <autostart.ini@gmail.com> <sgeto@ettercap-project.org>
#
# Permission is hereby granted, free of charge, to any person
# obtaining a copy of this software and associated documentation files
# (the "Software"), to deal in the Software without restriction,
# including without limitation the rights to use, copy, modify, merge,
# publish, distribute, sub-license, and/or sell copies of the Software,
# and to permit persons to whom the Software is furnished to do so,
# subject to the following conditions:
#
# The above copyright notice and this permission notice shall be
# included in all copies or substantial portions of the Software.
#
# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
# EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
# MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
# NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS
# BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN
# ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
# CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
# SOFTWARE.

# TODO
# ----
# - https://www.autoitconsulting.com/site/scripting/autoit-cmdlets-for-windows-powershell/
# - honor -reinstall

# Script entry-point arguments:
param(
    [switch]$debug = $false,
    [switch]$reinstall = $true
)

# Variables (and their Initial value)
# If any of these change, things may break.

# $PSScriptRoot isn't a thing when calling scripts from
# a remote location. That's why we use $WorkingDir instead.
$WorkingDir = Convert-Path .

$ChocoFlags = "--confirm",
              "--stoponfirstfailure",
              "--requirechecksum",
              "--allow-empty-checksums-secure",
              "--no-progress",
              "--limitoutput",
              "--cache-location=$WorkingDir"
$ChocoPKG   = "autoit.commandline" # add additional packages here
$AutoItPKG  = "autoit.commandline"
$AutoItPosh = "$env:ChocolateyInstall\lib\$AutoItPKG\tools\install\AutoItX\AutoItX.psd1"
$Setup      = "npcap-1.71.exe"

$SetupURL = "https://nmap.org/npcap/dist/"

$SetupFlags = "/disable_restore_point=yes",
              "/npf_startup=yes",
              "/loopback_support=yes",
              "/dlt_null=no",
              "/admin_only=no",
              "/dot11_support=yes",
              "/vlan_support=yes",
              "/winpcap_mode=yes",
              "/D=$WorkingDir"
$SetupTitle = "Npcap"

# Static Variables
# Probably no need to worry about them. Ever.
$SetupCopy  = "Insecure.Com LLC (`"The Nmap Project`")"
$SetupEULA  = "License Agreement"
$SetupLast  = "Installation Complete"
$Banner     = "`n
`tA Non-Interactive $SetupTitle Installation Script
`t___________________________________________`n
Copyright (C) 2018 Ali Abdulkadir <autostart.ini@gmail.com>`r
"
$Banner2    = "`r
$SetupTitle is proprietary to $SetupCopy.
No part of it may be redistributed, published or disclosed
except as outlined in the written contract supplied with
their product.`r
"

# Here we go...

function InitializeScriptEnvironment()
{
    # Get script start time
    $global:TimeStart = (Get-Date)

    # Check if we're on something headless
    if($env:CI)
    {
        $global:debug = $true # for now
        $global:reinstall = $true

        Write-Host $Banner -ForegroundColor White
    }
    else
    {
        # Clear the console
        Clear-Host;
        Write-Host $Banner -ForegroundColor White
        throw
    }

    if($debug)
    {
        Write-Host "[DEBUG MODE]`n" -ForegroundColor Red
        $poshver = ($PSVersionTable).PSVersion
        Write-Host "PowerShell Version: $poshver" -ForegroundColor Magenta
    }

    if(Get-Command "choco" -ErrorAction SilentlyContinue)
    {
        $global:have_choco = $true
        $choco_version = (Get-Command "choco").version
        if($debug)
        {
            Write-Host "Chocolatey Version: $choco_version" -ForegroundColor Magenta
        }
    } else {
        $global:have_choco = $false
        Write-Host "WARNING: This script uses Chocolatey, which was not found in your path." -ForegroundColor Red
    }

    $global:ErrorActionPreference = "Stop" # Stop script execution after any error.

}

function InstallPackage()
{
    if(!$env:ChocolateyInstall)
    {
        Write-Host "WARNING: Environment variable "ChocolateyInstall" not set" -ForegroundColor Red
        Write-Host "WARNING: This may not work..." -ForegroundColor Red
    }

    if($debug)
    {
        Write-Host "Installing needed packages via Chocolatey...`t`t" -ForegroundColor Cyan
        Write-Host "Chocolatey Flags: $ChocoFlags" -ForegroundColor Magenta
        Write-Host "Chocolatey Packages: $ChocoPKG" -ForegroundColor Magenta
        choco install $ChocoFlags $ChocoPKG
    } else {
        Write-Host "Installing needed packages via Chocolatey...`t`t" -NoNewline -ForegroundColor Cyan
        choco install $ChocoFlags $ChocoPKG | Out-Null
        @{$true = Write-Host "[SUCCESS]" -ForegroundColor Green}[$?]
    }
}

function ImportPoshModule()
{
    Write-Host "Importing $AutoItPKG PowerShell module...`t" -NoNewline -ForegroundColor Cyan
    Import-Module $AutoItPosh
    @{$true = Write-Host "[SUCCESS]" -ForegroundColor Green}[$?]
}

function DownloadFile([Parameter(Mandatory=$true)]$Link, [Parameter(Mandatory=$true)]$OutFile)
{
    Write-Host "Downloading $OutFile...`t`t" -NoNewline -ForegroundColor Cyan
    Invoke-WebRequest $Link -UseBasicParsing -OutFile $WorkingDir"\$OutFile"
    @{$true = Write-Host "[SUCCESS]" -ForegroundColor Green}[$?]
}

function RunSetup()
{
    $global:FileVersion = (Get-Item $WorkingDir"\$Setup").VersionInfo.FileVersion
    $global:ProductName = (Get-Item $WorkingDir"\$Setup").VersionInfo.ProductName

    if($debug)
    {
        Write-Host "Executing $Setup ($ProductName-$FileVersion)..." -ForegroundColor Magenta
        Write-Host "Install Flags: $SetupFlags" -ForegroundColor Magenta
        Invoke-AU3Run -Program $WorkingDir"\$Setup $SetupFlags"
    } else {
        Write-Host "Executing $Setup ($ProductName-$FileVersion)...`t" -NoNewline -ForegroundColor Cyan
        Invoke-AU3Run -Program $WorkingDir"\$Setup $SetupFlags" | Out-Null
        @{$true = Write-Host "[SUCCESS]" -ForegroundColor Green}[$?]
    }
}

function FocusSetup()
{
    Write-Host "Setting up handle to $ProductName setup window...`t`t" -NoNewline -ForegroundColor Cyan
    Wait-AU3Win -Title $SetupTitle | Out-Null
    $winHandle = Get-AU3WinHandle -Title $SetupTitle
    @{$true = Write-Host "[SUCCESS]" -ForegroundColor Green}[$?]

    if($debug)
    {
        Write-Host "Activating $ProductName window via handle..." -ForegroundColor Magenta
    }
    Show-AU3WinActivate -WinHandle $winHandle | Out-Null

    $controlHandle = Get-AU3ControlHandle -WinHandle $winhandle -Control "Static"

    if($reinstall)
    {
        if($debug)
        {
            Write-Host "(Just in case) Sending `"Yes`"...`t`t`t" -ForegroundColor Magenta
        }
        Send-AU3ControlKey -ControlHandle $controlHandle -Key "!y" -WinHandle $winHandle | Out-Null
    }

    Write-Host "Waiting for $ProductName $SetupEULA window...`t`t" -NoNewline -ForegroundColor Cyan
    Wait-AU3Win -Title $SetupTitle -Text $SetupEULA | Out-Null
    $winHandle = Get-AU3WinHandle -Title $SetupTitle
    @{$true = Write-Host "[SUCCESS]" -ForegroundColor Green}[$?]

    if($debug)
    {
        Write-Host "Activating $ProductName $SetupEULA window..." -ForegroundColor Magenta
    }
    Show-AU3WinActivate -WinHandle $winHandle | Out-Null
}

function NavigateSetup()
{
    $winHandle = Get-AU3WinHandle -Title $SetupTitle
    $controlHandle = Get-AU3ControlHandle -WinHandle $winhandle -Control "Static"

    if($debug)
    {
        Write-Host "Sending `"I Agree`"..." -ForegroundColor Magenta
    }
    Send-AU3ControlKey -ControlHandle $controlHandle -Key "!a" -WinHandle $winHandle | Out-Null

    if($debug)
    {
        Write-Host "Sending `"Install`"..." -ForegroundColor Magenta
    }
    Send-AU3ControlKey -ControlHandle $controlHandle -Key "!i" -WinHandle $winHandle | Out-Null

    Write-Host "Waiting for $ProductName setup controls to return...`t`t" -NoNewline -ForegroundColor Cyan
    Wait-AU3Win -Title $SetupTitle -Text $SetupLast | Out-Null
    $winHandle = Get-AU3WinHandle -Title $SetupTitle
    @{$true = Write-Host "[SUCCESS]" -ForegroundColor Green}[$?]

    if($debug)
    {
        Write-Host "Activating $ProductName Setup window..." -ForegroundColor Magenta
    }
    Show-AU3WinActivate -WinHandle $winHandle | Out-Null

    if($debug)
    {
        Write-Host "Sending `"Next`"..." -ForegroundColor Magenta
    }
    Send-AU3ControlKey -ControlHandle $controlHandle -Key "!n" -WinHandle $winHandle | Out-Null

    Write-Host "Finalizing $ProductName installation...`t`t`t" -NoNewline -ForegroundColor Cyan
    Send-AU3ControlKey -ControlHandle $controlHandle -Key "{ENTER}" -WinHandle $winHandle | Out-Null
    @{$true = Write-Host "[SUCCESS]" -ForegroundColor Green}[$?]
}

function ScriptCleanup()
{
    Write-Host "`rCleaning up...`t`t`t`t`t`t" -NoNewline -ForegroundColor Yellow
}

function ShowExecutionTime()
{
    $timeEnd = New-TimeSpan -Start $global:TimeStart -End $(Get-Date)
    Write-Host "Elapsed Time: $($timeEnd.Minutes) minute(s), $($timeEnd.Seconds) second(s)"
}

function main()
{
    InitializeScriptEnvironment;
    try
    {
        if($have_choco)
        {
            InstallPackage;
            ImportPoshModule;
        }
        DownloadFile -Link "$SetupURL$Setup" -OutFile "$Setup";
        RunSetup;
        FocusSetup;
        NavigateSetup;
    }
    finally
    {
        ScriptCleanup;
        ShowExecutionTime;
    }
}

# Entry point
main;
