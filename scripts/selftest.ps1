# Самопроверка RP Reel: прогоняет CLI на синтетическом видео и журнале.
# Тихо не проходит: любая ошибка - красный текст и выход 1.
$ErrorActionPreference = "Stop"
[Console]::OutputEncoding = [Text.Encoding]::UTF8

Write-Host "== Самопроверка RP Reel ==" -ForegroundColor Cyan

$root = Split-Path -Parent $PSScriptRoot
$cli = Join-Path $root "rp-reel-cli.exe"
if (-not (Test-Path $cli)) {
    Write-Host "НЕ НАЙДЕН: rp-reel-cli.exe (запускай из папки с релизом)" -ForegroundColor Red
    exit 1
}

# версия
& $cli -version
if ($LASTEXITCODE -ne 0) { throw "rp-reel-cli -version упал" }

$work = Join-Path ([System.IO.Path]::GetTempPath()) ("rp-reel-selftest-" + (Get-Random))
New-Item -ItemType Directory -Force -Path $work | Out-Null

# ffmpeg нужен для видео; если его нет - честно говорим
$ffmpeg = Get-Command ffmpeg -ErrorAction SilentlyContinue
if (-not $ffmpeg) {
    Write-Host "ffmpeg не найден в PATH: ядро проверено, нарезку не проверяю" -ForegroundColor Yellow
    Write-Host "Поставь ffmpeg и запусти самопроверку еще раз" -ForegroundColor Yellow
    Remove-Item $work -Recurse -Force -ErrorAction SilentlyContinue
    exit 0
}

Write-Host "делаю тестовое видео 120 секунд..."
& ffmpeg -y -loglevel error -f lavfi -i "testsrc=duration=120:size=320x240:rate=10" -f lavfi -i "sine=frequency=440:duration=120" -c:v libx264 -preset ultrafast -pix_fmt yuv420p -c:a aac (Join-Path $work "day.mp4")
if ($LASTEXITCODE -ne 0) { throw "ffmpeg не сделал тестовое видео" }

$base = [DateTimeOffset]::UtcNow.ToUnixTimeSeconds()
$lines = @(
    '{"ts":{0},"type":"kill","player":"Torpedo","other":"Banderas"}' -f ($base + 20)
    '{"ts":{0},"type":"wreck","player":"Banderas"}' -f ($base + 24)
    '{"ts":{0},"type":"arrest","player":"CopMike","other":"Robber"}' -f ($base + 70)
    '{"ts":{0},"type":"rob","player":"Shadow","other":"Bank"}' -f ($base + 100)
    "мусорная строка"
)
Set-Content -Path (Join-Path $work "rpreel.jsonl") -Value $lines -Encoding UTF8

Write-Host "запускаю нарезку..."
& $cli -log (Join-Path $work "rpreel.jsonl") -video (Join-Path $work "day.mp4") -video-ts $base -out (Join-Path $work "out")
if ($LASTEXITCODE -ne 0) { throw "rp-reel-cli упал на нарезке" }

$out = Join-Path $work "out"
$need = @("reel.mp4", "events.srt", "report.txt", "digest.json", "clip-001.mp4")
foreach ($f in $need) {
    $p = Join-Path $out $f
    if (-not (Test-Path $p)) { throw "результата нет: $f" }
    if ((Get-Item $p).Length -lt 100) { throw "результат пустой: $f" }
}

$digest = Get-Content (Join-Path $out "digest.json") -Raw -Encoding UTF8 | ConvertFrom-Json
if ($digest.on_video -ne 4) { throw "в дайджесте событий: $($digest.on_video), хочу 4" }
if ($digest.clips.Count -lt 2) { throw "клипов в дайджесте меньше 2" }

Write-Host "САМОПРОВЕРКА ПРОЙДЕНА: 4 события, $($digest.clips.Count) клипов, reel.mp4 готов" -ForegroundColor Green
Remove-Item $work -Recurse -Force -ErrorAction SilentlyContinue
