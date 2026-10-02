# TencentBlueKing is pleased to support the open source community by making
# 蓝鲸智云 - 服务治理 (BlueKing Service Governance) available.
# Copyright (C) Tencent. All rights reserved.
# Licensed under the MIT License (the "License"); you may not use this file except
# in compliance with the License. You may obtain a copy of the License at
#
#  http://opensource.org/licenses/MIT
#
# Unless required by applicable law or agreed to in writing, software distributed under
# the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
# either express or implied. See the License for the specific language governing permissions and
# limitations under the License.
#
# We undertake not to change the open source license (MIT license) applicable
# to the current version of the project delivered to anyone in the future.

[CmdletBinding()]
param(
    [string]$Version,
    [string]$InstallDir,
    [string]$BkmsBaseUrl,
    [string]$UpdateLatestUrl = $env:BKMS_CLI_UPDATE_LATEST_URL,
    [string]$UpdateDownloadUrlTemplate = $env:BKMS_CLI_UPDATE_DOWNLOAD_URL_TEMPLATE
)

# Keep functions and preferences local when invoked through irm | iex.
& {
    $ErrorActionPreference = 'Stop'
    $ProgressPreference = 'SilentlyContinue'
    $repository = 'TencentBlueKing/blueking-service-governance'
    # Internal distributions may set both defaults here; public distribution leaves them empty.
    $defaultUpdateLatestUrl = ''
    $defaultUpdateDownloadUrlTemplate = ''

    function Get-LatestVersion($LatestUrl) {
        $response = Invoke-WebRequest -UseBasicParsing -Uri $LatestUrl -TimeoutSec 30
        $latest = $response.Content.Trim()
        if ($latest -notmatch '^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$') {
            throw 'Invalid latest.txt; specify -Version X.Y.Z.'
        }
        return $latest
    }

    function Assert-HttpUrl($Address) {
        if ($Address -match '[{}\s]') { throw 'Update URLs cannot contain whitespace or unknown placeholders.' }
        $uri = [uri]$Address
        if (-not $uri.IsAbsoluteUri -or $uri.Scheme -notin @('http', 'https') -or -not $uri.Host -or $uri.Fragment) {
            throw 'Update URLs must be absolute HTTP(S) URLs without fragments.'
        }
    }

    function Expand-DownloadUrl($Template, $ReleaseVersion, $Archive) {
        return $Template.Replace('{version}', $ReleaseVersion).Replace('{archive}', $Archive)
    }

    function Confirm-Checksum($ArchivePath, $ChecksumsPath) {
        $name = [IO.Path]::GetFileName($ArchivePath)
        $pattern = '^([0-9a-fA-F]{64})\s+\*?' + [regex]::Escape($name) + '\s*$'
        $entries = @(Get-Content -LiteralPath $ChecksumsPath | Where-Object { $_ -match $pattern })
        if ($entries.Count -ne 1) { throw "Missing or duplicate checksum for $name." }
        $expected = [regex]::Match($entries[0], $pattern).Groups[1].Value
        $actual = (Get-FileHash -LiteralPath $ArchivePath -Algorithm SHA256).Hash
        if ($actual -ne $expected) { throw 'SHA-256 mismatch; existing installation was not changed.' }
    }

    function Install-Binary($ArchivePath, $TemporaryDir, $Destination) {
        $binary = Join-Path $TemporaryDir 'bkms-cli.exe'
        $zip = [IO.Compression.ZipFile]::OpenRead($ArchivePath)
        try {
            $entries = @($zip.Entries | Where-Object { $_.FullName -eq 'bkms-cli.exe' })
            if ($entries.Count -ne 1) { throw 'Archive must contain exactly one bkms-cli.exe.' }
            [IO.Compression.ZipFileExtensions]::ExtractToFile($entries[0], $binary)
        } finally {
            $zip.Dispose()
        }

        $versionProcess = Start-Process -FilePath $binary -ArgumentList 'version' -NoNewWindow -Wait -PassThru
        try {
            if ($versionProcess.ExitCode -ne 0) {
                throw 'Downloaded binary could not run; existing installation was not changed.'
            }
        } finally {
            $versionProcess.Dispose()
        }
        if (Test-Path -LiteralPath $Destination -PathType Container) {
            throw "$Destination is a directory."
        }

        # The temporary directory is on the destination filesystem.
        if ([IO.File]::Exists($Destination)) {
            [IO.File]::Replace($binary, $Destination, [NullString]::Value)
        } else {
            [IO.File]::Move($binary, $Destination)
        }
    }

    function Test-PathEntry([string]$PathValue, [string]$Directory) {
        $target = $Directory.Replace('/', '\').TrimEnd('\')
        foreach ($entry in ($PathValue -split ';')) {
            $expanded = [Environment]::ExpandEnvironmentVariables($entry.Trim().Trim('"'))
            if ($expanded.Replace('/', '\').TrimEnd('\') -ieq $target) { return $true }
        }
        return $false
    }

    function Add-UserPath($Directory) {
        if ($Directory -match '[;\r\n]') {
            Write-Host 'Cannot add this directory to PATH; choose an install directory without semicolons or newlines.'
            return
        }
        # Never persist $env:PATH: it also contains the machine PATH and session-only entries.
        try {
            $userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
            if (-not (Test-PathEntry $userPath $Directory)) {
                $updatedPath = if ($userPath) { "$Directory;$userPath" } else { $Directory }
                [Environment]::SetEnvironmentVariable('Path', $updatedPath, 'User')
            }
            if (-not (Test-PathEntry $env:PATH $Directory)) {
                $env:PATH = if ($env:PATH) { "$Directory;$env:PATH" } else { $Directory }
            }
            Write-Host "User PATH configured for $Directory. Restart other terminals to pick up the change."
        } catch {
            $quoted = $Directory.Replace("'", "''")
            Write-Host "Could not configure PATH automatically: $($_.Exception.Message)"
            Write-Host "Add $Directory to Path under your Windows user environment variables."
            Write-Host ("For this PowerShell session, run: " + '$env:PATH = ' + "'$quoted;' + " + '$env:PATH')
        }
    }

    if ($env:OS -ne 'Windows_NT') { throw 'This installer requires Windows; use install.sh on macOS/Linux.' }
    if ($PSVersionTable.PSVersion -lt [version]'5.1') { throw 'PowerShell 5.1 or newer is required.' }
    Add-Type -AssemblyName System.IO.Compression.FileSystem
    if (-not $UpdateLatestUrl) { $UpdateLatestUrl = $defaultUpdateLatestUrl }
    if (-not $UpdateDownloadUrlTemplate) { $UpdateDownloadUrlTemplate = $defaultUpdateDownloadUrlTemplate }
    if ([bool]$UpdateLatestUrl -xor [bool]$UpdateDownloadUrlTemplate) {
        throw 'Set both update URLs together.'
    }
    if ($UpdateLatestUrl) {
        $latestUrl = $UpdateLatestUrl
        $downloadTemplate = $UpdateDownloadUrlTemplate
    } else {
        $latestUrl = "https://raw.githubusercontent.com/$repository/main/bkms-cli/latest.txt"
        $downloadTemplate = "https://github.com/$repository/releases/download/bkms-cli%2Fv{version}/{archive}"
    }
    if (-not $downloadTemplate.Contains('{archive}')) { throw 'Download URL template must contain {archive}.' }
    Assert-HttpUrl $latestUrl
    Assert-HttpUrl (Expand-DownloadUrl $downloadTemplate '1.0.0' 'checksums.txt')

    $nativeArch = $env:PROCESSOR_ARCHITEW6432
    if (-not $nativeArch) { $nativeArch = $env:PROCESSOR_ARCHITECTURE }
    $arch = switch ($nativeArch) {
        'AMD64' { 'amd64' }
        'ARM64' { 'arm64' }
        default { throw "Unsupported architecture: $nativeArch" }
    }
    if (-not $InstallDir) {
        $InstallDir = Join-Path ([Environment]::GetFolderPath('LocalApplicationData')) 'bkms-cli\bin'
    }
    $InstallDir = $ExecutionContext.SessionState.Path.GetUnresolvedProviderPathFromPSPath($InstallDir)
    $temporaryDir = $null
    $originalProtocol = [Net.ServicePointManager]::SecurityProtocol

    try {
        [Net.ServicePointManager]::SecurityProtocol = $originalProtocol -bor [Net.SecurityProtocolType]::Tls12
        if (-not $Version) { $Version = Get-LatestVersion $latestUrl }
        $Version = $Version -replace '^bkms-cli/', '' -replace '^v', ''
        if ($Version -notmatch '^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$') {
            throw "Invalid version: $Version"
        }

        [IO.Directory]::CreateDirectory($InstallDir) | Out-Null
        $temporaryDir = Join-Path $InstallDir ('.bkms-cli.' + [guid]::NewGuid().ToString('N'))
        [IO.Directory]::CreateDirectory($temporaryDir) | Out-Null
        $archive = "bkms-cli_$($Version)_windows_$($arch).zip"
        $archivePath = Join-Path $temporaryDir $archive
        $checksumsPath = Join-Path $temporaryDir 'checksums.txt'
        $destination = Join-Path $InstallDir 'bkms-cli.exe'

        Write-Host "Installing bkms-cli $Version (windows/$arch)..."
        $archiveUrl = Expand-DownloadUrl $downloadTemplate $Version $archive
        $checksumsUrl = Expand-DownloadUrl $downloadTemplate $Version 'checksums.txt'
        Invoke-WebRequest -UseBasicParsing -Uri $archiveUrl -OutFile $archivePath -TimeoutSec 300
        Invoke-WebRequest -UseBasicParsing -Uri $checksumsUrl -OutFile $checksumsPath -TimeoutSec 30
        Confirm-Checksum $archivePath $checksumsPath
        Install-Binary $archivePath $temporaryDir $destination

        $configArgs = @('config', 'set', '--update-latest-url', $latestUrl, '--update-download-url-template', $downloadTemplate)
        if ($BkmsBaseUrl) { $configArgs += @('--bkms-base-url', $BkmsBaseUrl) }
        try {
            & $destination @configArgs
            if ($LASTEXITCODE -ne 0) { throw "config set exited with code $LASTEXITCODE." }
        } catch {
            Write-Warning "Endpoint configuration was not saved.$_" -WarningAction Continue
        }
        Write-Host "Installed to $destination"
        Add-UserPath $InstallDir
        $current = Get-Command bkms-cli -ErrorAction SilentlyContinue
        if ($current -and $current.Source -ne $destination) {
            Write-Host "PATH currently selects $($current.Source); adjust PATH to use this installation."
        }
    } finally {
        [Net.ServicePointManager]::SecurityProtocol = $originalProtocol
        if ($temporaryDir) {
            Remove-Item -LiteralPath $temporaryDir -Recurse -Force -ErrorAction SilentlyContinue
        }
    }
}
