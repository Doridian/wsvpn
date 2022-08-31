$OpenVPNMSI = "https://swupdate.openvpn.org/community/releases/OpenVPN-2.5.7-I602-amd64.msi"

$global:ErrorActionPreference = "Stop"

$WorkingDir = Convert-Path .

function DownloadFile([Parameter(Mandatory=$true)]$Link, [Parameter(Mandatory=$true)]$OutFile)
{
    Write-Host "Downloading $OutFile... "
    Invoke-WebRequest $Link -UseBasicParsing -OutFile $WorkingDir"\$OutFile"
    @{$true = Write-Host "[OK]"}[$?]
}

function MakeTAP([Parameter(Mandatory=$true)]$Name)
{
    Write-Host "Creating TAP $Name..."
    . "C:\Program Files\OpenVPN\bin\tapctl.exe" create --name "$Name"
    @{$true = Write-Host "[OK]"}[$?]

}

DownloadFile "$OpenVPNMSI" "openvpn.msi"

.\openvpn.msi /S /SELECT_SHURTCUTS=0 /SELECT_OPENVPN=0 /SELECT_SERVICE=0 /SELECT_TAP=1 /SELECT_OPENVPNGUI=0 /SELECT_ASSOCIATIONS=0 /SELECT_LAUNCH=0

MakeTAP "TAP0"
MakeTAP "TAP1"
MakeTAP "TAP2"
