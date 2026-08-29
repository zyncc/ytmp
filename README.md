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

To allow `yt-dlp` to fetch your personalized YouTube and YouTube Music playlists without login blocks, you must configure browser cookies.

Create or edit the configuration file at:
```bash
~/.config/yt-dlp/config
```

Add your browser cookie extraction configuration:

```conf
--cookies-from-browser chrome+gnomekeyring:~/.config/google-chrome
```

> [!TIP]
> If you are using a different browser or operating system, adjust the `--cookies-from-browser` flag accordingly:
> - **Brave**: `--cookies-from-browser brave`
> - **Firefox**: `--cookies-from-browser firefox`
> - **Chromium**: `--cookies-from-browser chromium`
> - **Edge**: `--cookies-from-browser edge`

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

### 🌐 Global Controls

| Key | Action |
|:---|:---|
| <kbd>Space</kbd> | Toggle Play / Pause |
| <kbd>.</kbd> | Next Track |
| <kbd>,</kbd> | Previous Track |
| <kbd>r</kbd> | Toggle Repeat Mode |
| <kbd>→</kbd> | Seek Forward 5 seconds |
| <kbd>←</kbd> | Seek Backward 5 seconds |
| <kbd>=</kbd> / <kbd>+</kbd> | Increase Volume |
| <kbd>-</kbd> | Decrease Volume |
| <kbd>q</kbd> | Toggle Queue Screen / Back |
| <kbd>Ctrl</kbd> + <kbd>c</kbd> | Quit Application |

---

### 📂 Playlists Screen

| Key | Action |
|:---|:---|
| <kbd>Enter</kbd> | Open selected playlist and load songs |
| <kbd>f</kbd> | Toggle Favorite status for selected playlist |
| <kbd>Ctrl</kbd> + <kbd>f</kbd> | Toggle Filter (Show All / Favorites Only) |
| <kbd>↑</kbd> / <kbd>k</kbd> | Move cursor up |
| <kbd>↓</kbd> / <kbd>j</kbd> | Move cursor down |

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
| <kbd>a</kbd> | Duplicate and add song to play next |
| <kbd>e</kbd> | Duplicate and enqueue song at end |
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
