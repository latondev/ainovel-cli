param(
    [string]$ConfigPath = ".ainovel/config.json",
    [string]$Provider = "",
    [string]$Prompt = "Reply with exactly: OK",
    [int]$TimeoutSec = 45,
    [switch]$AllProviders,
    [switch]$ShowResponse
)

$ErrorActionPreference = "Stop"

function Fail($Message) {
    Write-Error $Message
    exit 1
}

function Join-Url($BaseUrl, $Path) {
    return $BaseUrl.TrimEnd("/") + "/" + $Path.TrimStart("/")
}

function Get-JsonPropertyNames($Object) {
    if ($null -eq $Object) {
        return @()
    }
    return @($Object.PSObject.Properties | ForEach-Object { $_.Name })
}

if (-not (Test-Path -LiteralPath $ConfigPath)) {
    Fail "Config not found: $ConfigPath"
}

$config = Get-Content -Raw -LiteralPath $ConfigPath | ConvertFrom-Json
if ($null -eq $config.providers) {
    Fail "Config has no providers object: $ConfigPath"
}

$providerNames = Get-JsonPropertyNames $config.providers
if ($providerNames.Count -eq 0) {
    Fail "Config providers object is empty: $ConfigPath"
}

if ($Provider -ne "") {
    if ($providerNames -notcontains $Provider) {
        Fail "Provider '$Provider' not found in $ConfigPath"
    }
    $providersToTest = @($Provider)
} elseif ($AllProviders) {
    $providersToTest = $providerNames
} else {
    if ([string]::IsNullOrWhiteSpace($config.provider)) {
        Fail "Top-level provider is empty. Pass -Provider <name> or -AllProviders."
    }
    if ($providerNames -notcontains $config.provider) {
        Fail "Top-level provider '$($config.provider)' not found in providers."
    }
    $providersToTest = @($config.provider)
}

$results = New-Object System.Collections.Generic.List[object]

foreach ($providerName in $providersToTest) {
    $providerConfig = $config.providers.$providerName
    if ($null -eq $providerConfig) {
        $results.Add([pscustomobject]@{
            Provider = $providerName
            Model = "-"
            Status = "FAIL"
            Ms = 0
            Detail = "Provider config is null"
        })
        continue
    }

    $baseUrl = [string]$providerConfig.base_url
    $apiKey = [string]$providerConfig.api_key
    if ([string]::IsNullOrWhiteSpace($baseUrl)) {
        $results.Add([pscustomobject]@{
            Provider = $providerName
            Model = "-"
            Status = "FAIL"
            Ms = 0
            Detail = "Missing base_url"
        })
        continue
    }
    if ([string]::IsNullOrWhiteSpace($apiKey) -or $apiKey -match "YOUR_KEY|REPLACE|TODO|<") {
        $results.Add([pscustomobject]@{
            Provider = $providerName
            Model = "-"
            Status = "SKIP"
            Ms = 0
            Detail = "Missing or placeholder api_key"
        })
        continue
    }

    $models = @()
    if ($null -ne $providerConfig.models) {
        $models = @($providerConfig.models | Where-Object { -not [string]::IsNullOrWhiteSpace([string]$_) })
    }
    if ($models.Count -eq 0 -and $providerName -eq $config.provider -and -not [string]::IsNullOrWhiteSpace($config.model)) {
        $models = @([string]$config.model)
    }
    if ($models.Count -eq 0) {
        $results.Add([pscustomobject]@{
            Provider = $providerName
            Model = "-"
            Status = "SKIP"
            Ms = 0
            Detail = "No provider models and no matching top-level model"
        })
        continue
    }

    $uri = Join-Url $baseUrl "chat/completions"
    $headers = @{
        "Authorization" = "Bearer $apiKey"
        "Content-Type" = "application/json"
    }

    foreach ($model in $models) {
        $body = @{
            model = [string]$model
            messages = @(
                @{
                    role = "user"
                    content = $Prompt
                }
            )
            max_tokens = 16
            temperature = 0
        } | ConvertTo-Json -Depth 8

        $sw = [System.Diagnostics.Stopwatch]::StartNew()
        try {
            $resp = Invoke-RestMethod -Method Post -Uri $uri -Headers $headers -Body $body -TimeoutSec $TimeoutSec
            $sw.Stop()
            $text = ""
            if ($null -ne $resp.choices -and $resp.choices.Count -gt 0) {
                $text = [string]$resp.choices[0].message.content
            }
            if ([string]::IsNullOrWhiteSpace($text)) {
                $text = "HTTP OK, empty assistant content"
            }
            if (-not $ShowResponse -and $text.Length -gt 120) {
                $text = $text.Substring(0, 120) + "..."
            }
            $results.Add([pscustomobject]@{
                Provider = $providerName
                Model = [string]$model
                Status = "PASS"
                Ms = [int]$sw.ElapsedMilliseconds
                Detail = $text.Trim()
            })
        } catch {
            $sw.Stop()
            $detail = $_.Exception.Message
            if ($_.ErrorDetails -and $_.ErrorDetails.Message) {
                $detail = $_.ErrorDetails.Message
            }
            if ($detail.Length -gt 240) {
                $detail = $detail.Substring(0, 240) + "..."
            }
            $results.Add([pscustomobject]@{
                Provider = $providerName
                Model = [string]$model
                Status = "FAIL"
                Ms = [int]$sw.ElapsedMilliseconds
                Detail = $detail
            })
        }
    }
}

$results | Format-Table -AutoSize

$failed = @($results | Where-Object { $_.Status -eq "FAIL" })
if ($failed.Count -gt 0) {
    exit 2
}
