$OpenVPNMSI = "https://swupdate.openvpn.org/community/releases/OpenVPN-2.5.7-I602-amd64.msi"

$global:ErrorActionPreference = "Stop"

$WorkingDir = Convert-Path .

function DownloadFile([Parameter(Mandatory=$true)]$Link, [Parameter(Mandatory=$true)]$OutFile)
{
    Write-Host "Downloading $OutFile... "
    Invoke-WebRequest $Link -UseBasicParsing -OutFile "$WorkingDir\$OutFile"
    @{$true = Write-Host "[OK]"}[$?]
}

function MakeTAP([Parameter(Mandatory=$true)]$Name)
{
    Write-Host "Creating TAP $Name..."
    Start-Process "C:\Program Files\OpenVPN\bin\tapctl.exe" -ArgumentList "create --name `"$Name`"" -Wait
    @{$true = Write-Host "[OK]"}[$?]

}

DownloadFile "$OpenVPNMSI" "openvpn.msi"

Write-Host "Installing OpenVPN..."
Start-Process msiexec -ArgumentList "/i `"$WorkingDir\openvpn.msi`" ADDLOCAL=Drivers,Drivers.TAPWindows6,OpenVPN /quiet /norestart" -Wait
@{$true = Write-Host "[OK]"}[$?]

MakeTAP "TAP0"
MakeTAP "TAP1"
MakeTAP "TAP2"
