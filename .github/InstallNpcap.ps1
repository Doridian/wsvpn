$NPCAPURL = $env:NPCAP_URL

$global:ErrorActionPreference = "Stop"

$WorkingDir = Convert-Path .

function DownloadFile([Parameter(Mandatory=$true)]$Link, [Parameter(Mandatory=$true)]$OutFile)
{
    Write-Host "Downloading $OutFile... "
    Invoke-WebRequest $Link -UseBasicParsing -OutFile "$WorkingDir\$OutFile"
    @{$true = Write-Host "[OK]"}[$?]
}

DownloadFile "$NPCAPURL" "npcap.exe"

Write-Host "Installing NPCAP..."
Start-Process msiexec -ArgumentList "/i `"$WorkingDir\openvpn.msi`" ADDLOCAL=Drivers,Drivers.TAPWindows6,OpenVPN /quiet /norestart" -Wait
@{$true = Write-Host "[OK]"}[$?]

MakeTAP "TAP0"
MakeTAP "TAP1"
MakeTAP "TAP2"
