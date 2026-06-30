[CmdletBinding()]
param(
    [string]$ConfigPath = "$HOME\.ainovel\config.json",
    [string[]]$Models = @(),
    [int]$TimeoutSec = 60,
    [switch]$SkipStream
)

$ErrorActionPreference = "Stop"

function Read-JsonFile {
    param([string]$Path)

    if (-not (Test-Path -LiteralPath $Path)) {
        throw "Config file not found: $Path"
    }

    $raw = [System.IO.File]::ReadAllText($Path)
    $raw = $raw.TrimStart([char]0xFEFF)
    return $raw | ConvertFrom-Json
}

function Get-ErrorBody {
    param($ErrorRecord)

    if ($ErrorRecord.ErrorDetails -and $ErrorRecord.ErrorDetails.Message) {
        return $ErrorRecord.ErrorDetails.Message
    }

    $resp = $ErrorRecord.Exception.Response
    if ($resp -and $resp.GetResponseStream) {
        try {
            $reader = New-Object System.IO.StreamReader($resp.GetResponseStream())
            return $reader.ReadToEnd()
        }
        catch {
            return $ErrorRecord.Exception.Message
        }
    }

    return $ErrorRecord.Exception.Message
}

function Invoke-Curl {
    param(
        [string]$Method,
        [string]$Uri,
        [hashtable]$Headers,
        [object]$Body = $null,
        [int]$TimeoutSec = 60
    )

    $curlArgs = @(
        "-sS",
        "--ssl-no-revoke",
        "--max-time", [string]$TimeoutSec,
        "-w", "HTTP_STATUS:%{http_code}",
        "-X", $Method.ToUpperInvariant()
    )
    foreach ($key in $Headers.Keys) {
        $curlArgs += @("-H", "$key`: $($Headers[$key])")
    }

    if ($null -eq $Body) {
        $curlArgs += $Uri
    }
    else {
        $tmp = [System.IO.Path]::GetTempFileName()
        try {
            $json = $Body | ConvertTo-Json -Depth 20
            $enc = New-Object System.Text.UTF8Encoding($false)
            [System.IO.File]::WriteAllText($tmp, $json, $enc)
            $curlArgs += @("-H", "Content-Type: application/json", "--data-binary", "@$tmp", $Uri)
            Write-Verbose (($curlArgs | ForEach-Object {
                if ($_ -like "Authorization:*") { "Authorization: Bearer ***" } else { $_ }
            }) -join " ")
            $out = & curl.exe @curlArgs 2>&1 | Out-String
            return Convert-CurlOutput -Output $out
        }
        finally {
            Remove-Item -LiteralPath $tmp -ErrorAction SilentlyContinue
        }
    }

    Write-Verbose (($curlArgs | ForEach-Object {
        if ($_ -like "Authorization:*") { "Authorization: Bearer ***" } else { $_ }
    }) -join " ")
    $out = & curl.exe @curlArgs 2>&1 | Out-String
    return Convert-CurlOutput -Output $out
}

function Convert-CurlOutput {
    param([string]$Output)

    $marker = "HTTP_STATUS:"
    $idx = $Output.LastIndexOf($marker)
    if ($idx -lt 0) {
        throw $Output.Trim()
    }

    $body = $Output.Substring(0, $idx).Trim()
    $statusText = $Output.Substring($idx + $marker.Length).Trim()
    $status = 0
    [void][int]::TryParse($statusText, [ref]$status)

    if ($status -lt 200 -or $status -ge 300) {
        throw "HTTP $status`: $body"
    }

    if ([string]::IsNullOrWhiteSpace($body)) {
        return $null
    }

    try {
        return $body | ConvertFrom-Json
    }
    catch {
        return $body
    }
}

function Test-Chat {
    param(
        [string]$Model,
        [string]$BaseUrl,
        [hashtable]$Headers,
        [bool]$Stream,
        [int]$TimeoutSec
    )

    $body = @{
        model = $Model
        messages = @(
            @{
                role = "user"
                content = "Reply with OK only."
            }
        )
        stream = $Stream
    }

    if ($Model.ToLowerInvariant().StartsWith("gpt-5")) {
        $body.max_completion_tokens = 8
    }
    else {
        $body.max_tokens = 8
    }

    $mode = if ($Stream) { "stream" } else { "non-stream" }
    Write-Host ""
    Write-Host "== Chat test: $Model ($mode) =="

    try {
        $resp = Invoke-Curl -Method Post -Uri "$BaseUrl/chat/completions" -Headers $Headers -Body $body -TimeoutSec $TimeoutSec
        $content = if ($Stream) { ($resp | ConvertTo-Json -Depth 20) } else { $resp.choices[0].message.content }
        if ($content.Length -gt 700) {
            $content = $content.Substring(0, 700)
        }
        Write-Host "OK"
        Write-Host "Response: $content"
    }
    catch {
        Write-Host "FAILED"
        Write-Host $_.Exception.Message
    }
}

$cfg = Read-JsonFile -Path $ConfigPath
$providerKey = [string]$cfg.provider
if ([string]::IsNullOrWhiteSpace($providerKey)) {
    throw "Config field 'provider' is empty."
}

$provider = $cfg.providers.$providerKey
if ($null -eq $provider) {
    throw "Provider '$providerKey' is not configured in providers."
}

$apiKey = [string]$provider.api_key
$baseUrl = ([string]$provider.base_url).TrimEnd("/")
$type = [string]$provider.type
$defaultModel = [string]$cfg.model

Write-Host "Config: $ConfigPath"
Write-Host "Provider key: $providerKey"
Write-Host "Provider type: $type"
Write-Host "Base URL: $baseUrl"
Write-Host "Default model: $defaultModel"
Write-Host "API key present: $(-not [string]::IsNullOrWhiteSpace($apiKey))"

if ([string]::IsNullOrWhiteSpace($apiKey)) {
    throw "API key is missing in providers.$providerKey.api_key."
}
if ([string]::IsNullOrWhiteSpace($baseUrl)) {
    throw "Base URL is missing in providers.$providerKey.base_url."
}

$headers = @{
    Authorization = "Bearer $apiKey"
}

Write-Host ""
Write-Host "== GET /models =="
$listedModels = @()
try {
    $modelsResp = Invoke-Curl -Method Get -Uri "$baseUrl/models" -Headers $headers -TimeoutSec $TimeoutSec
    $listedModels = @($modelsResp.data | ForEach-Object { [string]$_.id } | Where-Object { $_ })
    if ($listedModels.Count -eq 0) {
        Write-Host "OK, but no models returned."
    }
    else {
        $listedModels | ForEach-Object { Write-Host "- $_" }
    }
}
catch {
    Write-Host "FAILED"
    Write-Host (Get-ErrorBody $_)
}

if ($Models.Count -eq 0) {
    $Models = @($defaultModel)
    foreach ($m in $provider.models) {
        if ($m -and $Models -notcontains [string]$m) {
            $Models += [string]$m
        }
    }
    foreach ($m in $listedModels) {
        if ($m -and $Models -notcontains $m) {
            $Models += $m
        }
    }
}

Write-Host ""
Write-Host "Models to test:"
$Models | ForEach-Object { Write-Host "- $_" }

foreach ($model in $Models) {
    Test-Chat -Model $model -BaseUrl $baseUrl -Headers $headers -Stream:$false -TimeoutSec $TimeoutSec
    if (-not $SkipStream) {
        Test-Chat -Model $model -BaseUrl $baseUrl -Headers $headers -Stream:$true -TimeoutSec $TimeoutSec
    }
}
