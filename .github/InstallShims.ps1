New-Item -Force -Path "$PROFILE.CurrentUserCurrentHost" -ItemType "file" -Value @"
function sudo(
        [switch]$E,
        [switch]$H,
        [parameter(ValueFromRemainingArguments = $true)]
        [string[]]$Passthrough
) {
    Invoke-Expression "& $Passthrough"
}

function which($Value) {
    Write-Output $Value
}
"@
