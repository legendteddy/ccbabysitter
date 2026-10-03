# Installs CC Babysitter on Windows, in Windows PowerShell 5.1 or PowerShell 7:
#
#   irm https://ccbabysitter.dev/install.ps1 | iex
#
# It downloads the release program for this machine, checks it against the
# release's checksums.txt, puts it in %LOCALAPPDATA%\Programs\CCBabysitter
# and adds that folder to your user Path. It needs no admin rights.
#
# Settings, read from the environment:
#   CCBABYSITTER_VERSION       release tag to install, such as v0.4.0
#                              (default: the latest release)
#   CCBABYSITTER_INSTALL_DIR   folder to install into
#                              (default: %LOCALAPPDATA%\Programs\CCBabysitter)
#   CCBABYSITTER_DOWNLOAD_URL  base URL holding the release files
#                              (default: the GitHub release for the version)
#
# Everything is in one function called on the last line, so a download cut
# short runs nothing at all, and the settings it changes stay inside it
# rather than in the PowerShell window it runs in. A failure ends in a
# thrown error rather than exit, which would close that window, so a
# script or tool that ran this sees it failed.

function Install-CCBabysitter {
    $ErrorActionPreference = 'Stop'
    # The progress bar makes downloads many times slower in 5.1.
    $ProgressPreference = 'SilentlyContinue'

    $repoUrl = 'https://github.com/pejmanebrahimi/ccbabysitter'
    $runningMessage = 'CC Babysitter is running. Quit it (Ctrl+C in its window), then run this again.'

    # Test-FileLocked reports whether an exception, or one inside it, is a
    # sharing or lock violation: the file is open in another program.
    function Test-FileLocked($exception) {
        for ($e = $exception; $e; $e = $e.InnerException) {
            $code = $e.HResult -band 0xFFFF
            if ($code -eq 32 -or $code -eq 33) {
                return $true
            }
        }
        return $false
    }

    # Get-Reason is the innermost message of an exception, which is the
    # one that says what went wrong rather than which call failed.
    function Get-Reason($exception) {
        while ($exception.InnerException) {
            $exception = $exception.InnerException
        }
        return $exception.Message
    }

    # Windows PowerShell 5.1 may not offer TLS 1.2, which GitHub requires.
    if ($PSVersionTable.PSVersion.Major -lt 6) {
        [Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12
    }

    $machine = $env:PROCESSOR_ARCHITEW6432
    if (-not $machine) {
        $machine = $env:PROCESSOR_ARCHITECTURE
    }
    switch ($machine) {
        'AMD64' { $arch = 'amd64' }
        'ARM64' { $arch = 'arm64' }
        default {
            if (-not $machine) {
                $machine = 'unknown'
            }
            throw "There is no CC Babysitter build for the $machine processor. Builds exist for AMD64 and ARM64."
        }
    }
    $asset = "ccbabysitter-windows-$arch.exe"

    if ($env:CCBABYSITTER_DOWNLOAD_URL) {
        $base = $env:CCBABYSITTER_DOWNLOAD_URL.TrimEnd('/')
    } else {
        $version = $env:CCBABYSITTER_VERSION
        if (-not $version -or $version -eq 'latest') {
            $base = "$repoUrl/releases/latest/download"
        } else {
            # A tag is v0.4.0; 0.4.0 and V0.4.0 mean the same one.
            if ($version -match '^v') {
                $version = $version.Substring(1)
            }
            $base = "$repoUrl/releases/download/v$version"
        }
    }

    if ($env:CCBABYSITTER_INSTALL_DIR) {
        $dir = $env:CCBABYSITTER_INSTALL_DIR
    } else {
        $dir = Join-Path $env:LOCALAPPDATA 'Programs\CCBabysitter'
    }
    # A relative folder is taken from this window's current location.
    $dir = $ExecutionContext.SessionState.Path.GetUnresolvedProviderPathFromPSPath($dir)
    if ($dir.Length -gt 3) {
        $dir = $dir.TrimEnd('\')
    }
    $dest = Join-Path $dir 'ccbabysitter.exe'

    $tmp = Join-Path ([IO.Path]::GetTempPath()) ('ccbabysitter-' + [Guid]::NewGuid().ToString('N'))
    New-Item -ItemType Directory -Path $tmp -Force | Out-Null
    $part = $null
    $bak = "$dest.bak"
    $moved = $false
    try {
        $file = Join-Path $tmp $asset
        $sums = Join-Path $tmp 'checksums.txt'
        Write-Host "Downloading $asset from $base"
        foreach ($pair in @(@("$base/$asset", $file), @("$base/checksums.txt", $sums))) {
            try {
                Invoke-WebRequest -UseBasicParsing -Uri $pair[0] -OutFile $pair[1]
            } catch {
                throw "Could not download $($pair[0]): $(Get-Reason $_.Exception)"
            }
        }

        $want = $null
        foreach ($line in Get-Content -LiteralPath $sums) {
            $fields = $line.Trim() -split '\s+'
            if ($fields.Count -ge 2 -and ($fields[1] -ceq $asset -or $fields[1] -ceq "*$asset")) {
                $want = $fields[0].ToLowerInvariant()
                break
            }
        }
        if (-not $want) {
            throw "checksums.txt has no line for $asset, so the download cannot be checked. Nothing was installed."
        }
        $got = (Get-FileHash -LiteralPath $file -Algorithm SHA256).Hash.ToLowerInvariant()
        if ($got -ne $want) {
            throw "The download of $asset does not match its checksum (expected $want, got $got). Nothing was installed."
        }

        # The new copy is written beside the old one under a temporary
        # name. The old ccbabysitter.exe is then moved aside to
        # ccbabysitter.exe.bak and the new copy moved into place; if that
        # move fails, the old copy is moved back. The .bak is kept until
        # the installed program has run (see below), so a failed install
        # never loses the old program or leaves a broken
        # ccbabysitter.exe behind.
        New-Item -ItemType Directory -Path $dir -Force | Out-Null
        $part = Join-Path $dir ('.ccbabysitter-' + [Guid]::NewGuid().ToString('N') + '.part')
        try {
            [IO.File]::Copy($file, $part, $true)
            if (Test-Path -LiteralPath $dest) {
                # Windows keeps a running program's file open, so it can
                # be neither written nor moved. Opening it for writing
                # finds that out while the old copy is still whole.
                [IO.File]::Open($dest, [IO.FileMode]::Open, [IO.FileAccess]::ReadWrite, [IO.FileShare]::None).Close()
                if (Test-Path -LiteralPath $bak) {
                    [IO.File]::Delete($bak)
                }
                [IO.File]::Move($dest, $bak)
                $moved = $true
            }
            try {
                [IO.File]::Move($part, $dest)
            } catch {
                $failure = $_
                if ($moved) {
                    try {
                        [IO.File]::Move($bak, $dest)
                        $moved = $false
                    } catch {
                        throw "$(Get-Reason $failure.Exception). The old copy could not be put back and is at $bak"
                    }
                }
                throw $failure
            }
        } catch {
            if (Test-FileLocked $_.Exception) {
                throw $runningMessage
            }
            throw "Could not install ${dest}: $(Get-Reason $_.Exception)"
        }
    } finally {
        if ($part -and (Test-Path -LiteralPath $part)) {
            Remove-Item -LiteralPath $part -Force -ErrorAction SilentlyContinue
        }
        Remove-Item -LiteralPath $tmp -Recurse -Force -ErrorAction SilentlyContinue
    }

    # The old copy is deleted only now that the new one has run. If it
    # does not run, the old one is put back as well as it can be.
    $installed = $null
    $runs = $false
    try {
        $installed = & $dest version
        $runs = ($LASTEXITCODE -eq 0)
    } catch {
        $runs = $false
    }
    if (-not $runs) {
        if ($moved) {
            try {
                [IO.File]::Delete($dest)
                [IO.File]::Move($bak, $dest)
            } catch {
                throw "$dest was installed but does not run on this machine, and the old copy could not be put back. It is at $bak"
            }
            throw "$dest was installed but does not run on this machine, so the old copy was put back."
        }
        throw "$dest was installed but does not run on this machine."
    }
    if ($moved) {
        Remove-Item -LiteralPath $bak -Force -ErrorAction SilentlyContinue
    }
    Write-Host "Installed $installed to $dest"

    # The user Path is read and written as stored, so entries such as
    # %USERPROFILE%\bin keep their variables rather than being expanded.
    $key = [Microsoft.Win32.Registry]::CurrentUser.CreateSubKey('Environment')
    try {
        $userPath = [string]$key.GetValue('Path', '', [Microsoft.Win32.RegistryValueOptions]::DoNotExpandEnvironmentNames)
        $entries = @($userPath -split ';' | Where-Object { $_ } | ForEach-Object { [Environment]::ExpandEnvironmentVariables($_).TrimEnd('\') })
        if ($entries -notcontains $dir) {
            if ($userPath) {
                $newPath = $userPath.TrimEnd(';') + ';' + $dir
            } else {
                $newPath = $dir
            }
            $key.SetValue('Path', $newPath, [Microsoft.Win32.RegistryValueKind]::ExpandString)
            # Setting a user variable this way tells Windows the
            # environment changed, so windows opened from now on see the
            # new Path; this one is set and removed only for that.
            [Environment]::SetEnvironmentVariable('CCBABYSITTER_INSTALL_REFRESH', '1', 'User')
            [Environment]::SetEnvironmentVariable('CCBABYSITTER_INSTALL_REFRESH', $null, 'User')
            Write-Host "Added $dir to your user Path."
        }
    } finally {
        $key.Close()
    }
    $sessionEntries = @($env:Path -split ';' | Where-Object { $_ } | ForEach-Object { $_.TrimEnd('\') })
    if ($sessionEntries -notcontains $dir) {
        $env:Path = $env:Path.TrimEnd(';') + ';' + $dir
        Write-Host "Added $dir to the Path of this window."
    }

    Write-Host ''
    Write-Host 'Start it with: ccbabysitter'
}

Install-CCBabysitter
