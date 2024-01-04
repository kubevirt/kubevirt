$ErrorActionPreference = 'Stop'
$version               = 'v1.1.0'
$arch                  = if ($env:ChocolateyForceX86 -eq 'true') {'32'} else {'64'}
$url                   = "https://github.com/kubevirt/kubevirt/releases/download/${version}/virtctl-${version}-windows-amd${arch}.exe"
$checksum              = '52d27e0e1b553705b08d4c29c9518893c855e4408cfe1a2bbc6ed2f385b39d7e' 
$checksumType          = 'sha512'
$File                  = Join-Path (Join-Path $env:ChocolateyInstall (Join-Path 'lib' $env:ChocolateyPackageName)) 'virtctl.exe'

$file = Get-ChocolateyWebFile -PackageName $env:ChocolateyPackageName `
 -FileFullPath $File `
 -Url64bit $url  `
 -CheckSum $checksum `
 -CheckSumType $checksumType `
 -CheckSum64 $checksum64 `
 -CheckSumType64 $checksumType64
