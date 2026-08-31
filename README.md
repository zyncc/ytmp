<div align="center">

# 🎵 ytmp

### A fast, minimal, and beautiful YouTube & YouTube Music TUI player written in Go.

[![Go Version](https://img.shields.io/badge/Go-1.27+-00ADD8?style=flat&logo=go)](https://golang.org)
[![Bubble Tea](https://img.shields.io/badge/Charm-Bubble_Tea_v2-F25D94?style=flat)](https://charm.sh)
[![MPV](https://img.shields.io/badge/Audio-mpv-7B2FBE?style=flat&logo=mpv)](https://mpv.io)
[![yt-dlp](https://img.shields.io/badge/Extractor-yt--dlp-red?style=flat&logo=youtube)](https://github.com/yt-dlp/yt-dlp)
[![License](https://img.shields.io/badge/License-MIT-blue.svg?style=flat)](LICENSE)

<p align="center">
  <a href="#features">Features</a> •
  <a href="#prerequisites">Prerequisites</a> •
  <a href="#installation">Installation</a> •
  <a href="#yt-dlp-configuration">Authentication & Setup</a> •
  <a href="#keybindings">Keybindings</a> •
  <a href="#configuration">Configuration</a> •
  <a href="#known-edge-cases">Known Edge Cases</a>
</p>

</div>

---

## 🌟 Overview

**ytmp** is a high-performance terminal music player designed to bring your YouTube and YouTube Music playlists directly into your command line. Built with [Charm's Bubble Tea v2](https://github.com/charmbracelet/bubbletea) for a responsive TUI, [mpv](https://mpv.io) for robust audio streaming, and [yt-dlp](https://github.com/yt-dlp/yt-dlp) for metadata extraction, **ytmp** delivers a distraction-free and lightweight listening experience.

---

## ✨ Features

- **⚡ Gapless & Low-Latency Playback**  
  Asynchronously prefetches stream URLs for subsequent and preceding tracks in the background, eliminating buffering delays between songs.
- **💾 Dual-Tier Caching Architecture**  
  - **Persistent SQLite Database (`~/.cache/ytmp/cache.db`)**: Automatically caches playlist and song metadata (titles, artists, durations, view counts, thumbnails) for instantaneous startup and offline navigation.
  - **In-Memory Stream Hash Map**: Caches direct audio stream URLs in memory to prevent repetitive network lookups during a session.
- **🔀 Flawless Queue Management**  
  Interactive queue screen with support for instant shuffling, "Play Next" priority insertion, appending songs on the fly, and seamless cursor tracking.
- **❤️ Favorites & Playlist Filtering**  
  Mark favorite playlists with a single keypress and toggle between viewing all playlists or only your starred favorites.
- **🎛️ Headless MPV IPC Engine**  
  Direct socket-based IPC communication with `mpv` allows smooth volume adjustments, real-time scrubbing/seeking, and event listening.
- **🛰️ Full MPRIS 2.2 & playerctl Integration**  
  Seamlessly control playback, scrub tracks, adjust volume, and view metadata through media keys, status bars (Waybar, Polybar), desktop widgets, or `playerctl`.
- **🎨 Configurable & Minimal TUI**  
  Clean aesthetic with customizable ANSI themes, dynamic progress bars, duration indicators, and automatic terminal resizing.

---

## 📋 Prerequisites

Before running **ytmp**, ensure the following tools are installed on your system:

1. **[mpv](https://mpv.io)** — Command-line media player used as the playback engine.
   ```bash
   # Arch Linux
   sudo pacman -S mpv

   # Debian / Ubuntu
   sudo apt install mpv

   # macOS (Homebrew)
   brew install mpv
   ```

2. **[yt-dlp](https://github.com/yt-dlp/yt-dlp)** — YouTube metadata and audio stream extractor.
   ```bash
   # Arch Linux
   sudo pacman -S yt-dlp

   # Debian / Ubuntu
   sudo apt install yt-dlp

   # macOS (Homebrew)
   brew install yt-dlp

   # Or via pip / pipx
   pipx install yt-dlp
   ```

---

## 🔐 yt-dlp Configuration (Authentication)

To allow `yt-dlp` (and **ytmp**) to fetch your personalized YouTube and YouTube Music playlists without login blocks or bot detection issues, you must configure authentication using browser cookies.

Create or edit the configuration file at:
```bash
mkdir -p ~/.config/yt-dlp
nano ~/.config/yt-dlp/config
```

Choose one of the two authentication methods below:

### Option 1: Exported `cookies.txt` File (Recommended)

This is the most reliable method and avoids browser database locking or keyring decryption issues:

1. Install a browser extension that exports cookies in Netscape format (e.g., [Get cookies.txt LOCALLY](https://chromewebstore.google.com/detail/get-cookiestxt-locally/cclelndahbngbenkjcffliehddfacjne) for Chrome/Chromium or Firefox).
2. Log in to [YouTube](https://www.youtube.com) and [YouTube Music](https://music.youtube.com).
3. Export your cookies and save the file to `~/.config/yt-dlp/cookies.txt`.
4. Update `~/.config/yt-dlp/config` to use the cookies file:

```conf
--cookies ~/.config/yt-dlp/cookies.txt
```

> [!IMPORTANT]
> Keep your `cookies.txt` file secure and private, as it contains active session credentials.

---

### Option 2: Direct Browser Extraction

Alternatively, `yt-dlp` can attempt to automatically extract cookies directly from your installed browser:

```conf
--cookies-from-browser chrome
```

**Supported browsers**:
- **Chrome**: `--cookies-from-browser chrome`
- **Firefox**: `--cookies-from-browser firefox`
- **Brave**: `--cookies-from-browser brave`
- **Chromium**: `--cookies-from-browser chromium`
- **Edge**: `--cookies-from-browser edge`

> [!TIP]
> **Linux & Keyring Note**:  
> On Linux, Chromium-based browsers (Chrome, Brave, Chromium, Edge) encrypt stored cookies using the system keyring. It may be necessary to specify your keyring (e.g., `+gnomekeyring` or `+kwallet5`/`+kwallet6`) and profile path:
> ```conf
> --cookies-from-browser chrome+gnomekeyring:~/.config/google-chrome
> # Or for Brave:
> --cookies-from-browser brave+gnomekeyring:~/.config/BraveSoftware/Brave-Browser
> ```

---

## 🚀 Installation

### From Source

Make sure you have **Go 1.27+** installed:

```bash
# Clone the repository
git clone https://github.com/zyncc/ytmp.git
cd ytmp

# Build binary
go build -o ytmp ./cmd/ytmp

# (Optional) Move to your PATH
sudo mv ytmp /usr/local/bin/
```

### Go Install

```bash
go install github.com/zyncc/ytmp/cmd/ytmp@latest
```

---

## ⌨️ Keybindings

### 🎵 Playback & Audio

| Key | Action |
|:---|:---|
| <kbd>Space</kbd> | Toggle Play / Pause |
| <kbd>.</kbd> | Next track |
| <kbd>,</kbd> | Previous track / Restart track (if > 10s elapsed) |
| <kbd>→</kbd> | Seek forward 5 seconds |
| <kbd>←</kbd> | Seek backward 5 seconds |
| <kbd>+</kbd> / <kbd>=</kbd> | Increase volume |
| <kbd>-</kbd> | Decrease volume |
| <kbd>r</kbd> | Toggle repeat mode |

---

### 🧭 Navigation & Scrolling

| Key | Action |
|:---|:---|
| <kbd>↑</kbd> / <kbd>k</kbd> | Move cursor up |
| <kbd>↓</kbd> / <kbd>j</kbd> | Move cursor down |
| <kbd>g</kbd> / <kbd>Home</kbd> | Jump to top of list |
| <kbd>G</kbd> / <kbd>End</kbd> | Jump to bottom of list |
| <kbd>Ctrl</kbd> + <kbd>u</kbd> / <kbd>PgUp</kbd> | Scroll page up |
| <kbd>Ctrl</kbd> + <kbd>d</kbd> / <kbd>PgDown</kbd> | Scroll page down |

---

### 🌐 General & Application

| Key | Action |
|:---|:---|
| <kbd>q</kbd> | Toggle Queue screen / Back |
| <kbd>?</kbd> | Toggle Keybinds help overlay |
| <kbd>Esc</kbd> | Back / Return to previous view |
| <kbd>Ctrl</kbd> + <kbd>c</kbd> | Quit application |

---

### 📂 Playlists Screen

| Key | Action |
|:---|:---|
| <kbd>Enter</kbd> | Open selected playlist and load songs |
| <kbd>f</kbd> | Toggle Favorite status for selected playlist |
| <kbd>Ctrl</kbd> + <kbd>f</kbd> | Toggle Filter (Show All / Favorites Only) |

---

### 🎵 Songs Screen

| Key | Action |
|:---|:---|
| <kbd>Enter</kbd> | Play song immediately and load remaining playlist into queue |
| <kbd>s</kbd> | Shuffle all songs in current playlist and start playback |
| <kbd>a</kbd> | Add song to play next (inserted immediately after current track) |
| <kbd>e</kbd> | Enqueue song (append to the end of queue) |
| <kbd>Esc</kbd> | Return to Playlists screen |

---

### 📑 Queue Screen

| Key | Action |
|:---|:---|
| <kbd>Enter</kbd> | Jump to and play selected song |
| <kbd>d</kbd> | Remove selected song from queue |
| <kbd>a</kbd> | Duplicate and add song to play next |
| <kbd>e</kbd> | Duplicate and enqueue song at end of queue |
| <kbd>Esc</kbd> / <kbd>q</kbd> | Return to previous screen |

---

## ⚙️ Configuration

`ytmp` automatically generates a default configuration file upon initial launch at:

```
~/.config/ytmp/config.toml
```

### Default `config.toml`

```toml
[general]
toggle_favorites = false

[player]
volume = 100
volume_increment_amount = 5

[theme]
primary = "4"       # Primary accent color (ANSI or Hex)
secondary = "6"     # Secondary highlight color
text = "7"          # Standard text
subtle = "8"        # Subtle / dimmed labels and borders
border = "0"        # Table borders
success = "2"       # Success indicator
error = "1"         # Error indicator
```

---

## 🗄️ Cache & Storage

- **Database**: Metadata is cached locally in SQLite at `~/.cache/ytmp/cache.db` with automated migrations.
- **Logs**: Production runtime logs are written for diagnostics and debugging.

To reset the database cache, simply remove the cache directory:
```bash
rm -rf ~/.cache/ytmp
```

---

## ⚠️ Known Edge Cases & Disclaimers

> [!WARNING]
> **Stream URL Expiration (6-Hour YouTube CDN Limit)**:  
> YouTube's direct audio stream URLs (`googlevideo.com`) are cryptographically signed with a time-to-live (TTL) of **approximately 6 hours**. 
> 
> Currently, `ytmp` caches these stream URLs in an in-memory map for instantaneous and gapless track switching. If you leave `ytmp` running continuously for more than 6 hours and attempt to play a track whose URL was cached at the start of the session, playback may fail due to URL expiration. A background token refresh mechanism is planned for a future update.

---

## 🛠️ Tech Stack

- **Language**: [Go](https://go.dev/)
- **TUI Framework**: [Bubble Tea v2](https://charm.sh/), [Lipgloss v2](https://github.com/charmbracelet/lipgloss), [Bubbles v2](https://github.com/charmbracelet/bubbles)
- **Audio Engine**: Headless [mpv](https://mpv.io) via Unix Domain Socket IPC
- **Metadata Extraction**: [yt-dlp](https://github.com/yt-dlp/yt-dlp)
- **Database**: Pure Go SQLite ([modernc.org/sqlite](https://gitlab.com/cznic/sqlite)) + [golang-migrate](https://github.com/golang-migrate/migrate)

---

## 📄 License

This project is licensed under the [MIT License](LICENSE).
