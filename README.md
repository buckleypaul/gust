# gust

[![CI](https://github.com/buckleypaul/gust/actions/workflows/ci.yml/badge.svg)](https://github.com/buckleypaul/gust/actions/workflows/ci.yml)
![Go 1.22](https://img.shields.io/badge/Go-1.22-00ADD8?logo=go)

A terminal UI for [Zephyr RTOS](https://zephyrproject.org) development. Build, flash, test, and monitor your firmware from a single interface without leaving the terminal.

<!-- screenshot: main view showing sidebar + build page -->

## What it does

Nine pages accessible from a sidebar:

| Page | Purpose |
|------|---------|
| **Workspace** | West workspace health and `west update` |
| **Build** | Build firmware for any board |
| **Flash** | Flash to connected hardware |
| **Test** | Run west test suites |
| **Monitor** | Serial console with send/receive |
| **Artifacts** | History of builds, flashes, tests, and serial logs |
| **West** | Run arbitrary west commands |
| **Config** | Browse and search Kconfig symbols from `prj.conf` |
| **Settings** | Edit default board, serial port, baud rate, and more |

<!-- screenshot: serial monitor page -->

## Installation

**From source** (requires Go 1.22+):

```bash
git clone https://github.com/buckleypaul/gust.git
cd gust
make install
```

## Usage

Run from inside a Zephyr workspace:

```bash
cd ~/zephyr-workspace
gust
```

`Tab` switches focus between the sidebar and the active page. Number keys `1`–`9` jump directly to any page. `q` quits.
