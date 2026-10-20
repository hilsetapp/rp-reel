[**Русский**](README.md) | **English**

# RP Reel

Cut RP-server highlights from a video recording. A server plugin writes an event log (kills, wrecks, arrests, robberies), the app places events on the video timeline and exports clips with one click - losslessly, even for 4K.

![event timeline on video](assets/screenshot.png)

## What it is for

- server admins posting a "moment of the day" to Discord/VK/Telegram;
- streamers assembling a recap from an evening recording;
- admins reviewing an incident: click a marker, see the frame before the event.

## How it works

1. **Server plugin** writes `rpreel.jsonl`: time, event type, players. Ready includes for [SA-MP](plugins/samp/rpreel.inc), [RAGE MP](plugins/ragemp/reel.js) and [FiveM](plugins/fivem/reel.lua), about 20 lines each.
2. **Record video** with OBS on the same machine as the server, so the clocks match. Different machines? There is a manual offset.
3. **Timeline**: the app reads the log and the video, draws a strip of thumbnails with colored event markers, plus stats on the right: top players, event types, hottest minute.
4. **Export**: ffmpeg cuts windows around events (`-c copy`, no re-encoding) and concatenates them into `reel.mp4`. Also writes `events.srt` with captions, `report.txt` and `digest.json`.

## Install

1. Download the [latest release](https://github.com/hilsetapp/rp-reel/releases/latest/download/rp-reel-windows.zip) and unpack it anywhere.
2. Install [ffmpeg](https://ffmpeg.org/download.html) and add it to PATH (the app tells you loudly if it is missing).
3. Run `rp-reel.exe`.

Windows 10/11. No installation, no registry writes, no network calls: everything happens on your machine.

## Quick start

1. On the server: `#include <rpreel>` in the gamemode, `Reel_Open()` in `OnGameModeInit`, `Reel_Kill(killerid, victimid, reason)` in `OnPlayerDeath` (SA-MP; RAGE MP and FiveM ship ready subscriptions).
2. Record the evening in OBS, copy `rpreel.jsonl` to your machine.
3. In the app: "Видео..." (video) and "Журнал..." (log) buttons, then "Собрать таймлайн" (build timeline).
4. Click a clip to see the events inside; "Показать в плеере" opens VLC at that second.
5. "Вырезать ролик" (export) - done, the output folder opens itself.

Command line also works:

```
rp-reel.exe -video day.mp4 -log rpreel.jsonl
```

## Time sync

If markers do not line up with the video:
- **"Начало записи, Unix"** - recording start time in Unix seconds;
- **"Сдвиг видео, сек"** - nudge the video against the log.

By default the first event marks the start of the video.

## Preset gallery

A preset is the window around an event plus a clip limit. Built in:

| Preset | Window | Clips | For |
| --- | --- | --- | --- |
| Base | 10+10 s | all | everyday cuts |
| Twitch Shorts | 5+7 s | 12 | short vertical reels |
| Discord digest | 8+12 s | 40 | the whole evening in one file |
| VK vertical 9:16 | 6+8 s | 10 | VK Clips and Reels (re-encodes) |

Your own preset: a json file into the `templates` folder next to the exe, format in [`internal/template/presets/README.md`](internal/template/presets/README.md).

## Command line

```
rp-reel-cli -log rpreel.jsonl -video day.mp4 -video-ts 1729440000
rp-reel-cli -log rpreel.jsonl -video day.mp4 -preset discord-digest
rp-reel-cli -log rpreel.jsonl -video day.mp4 -no-cut   # report and plan only
```

Flags: `-log` event log, `-video` recording, `-video-ts` recording start (Unix, s), `-offset` video shift (s), `-preset` preset, `-out` output dir (default `reel-out`), `-no-cut` skip ffmpeg cutting, `-version`.

## Inside

- Go + Gio: a single exe, no runtime;
- `ffmpeg -c copy` cutting: keyframes, instant, no GPU load;
- timeline thumbnails are made by ffmpeg itself;
- privacy: logs and video never leave your machine.

## Roadmap

- event map from log coordinates;
- auto-sync by audio (shots, explosions);
- ready resources for MTA and Oxen servers.

## License

MIT. ffmpeg is required separately, its license is [here](THIRD_PARTY_NOTICES.md).

## Contact

Issues are read, RU replies welcome. Server plugin ideas and presets accepted.
