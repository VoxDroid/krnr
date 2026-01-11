$packageName = 'krnr'
$toolsDir = "$(Split-Path -Parent $MyInvocation.MyCommand.Definition)"
$url = 'REPLACE_URL_TO_MSI'
$checksum = 'REPLACE_SHA256'
$checksumType = 'sha256'

Install-ChocolateyPackage $packageName 'msi' $silentArgs $url -Checksum $checksum -ChecksumType $checksumType
