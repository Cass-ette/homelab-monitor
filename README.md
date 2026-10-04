# Homelab Monitor

<div align="center">

![Homelab Monitor](https://img.shields.io/badge/Go-1.23-00ADD8?style=flat&logo=go)
![License](https://img.shields.io/badge/license-MIT-green)
![Platform](https://img.shields.io/badge/platform-Linux%20%7C%20Windows%20%7C%20macOS-lightgrey)

**Lightweight, beautiful monitoring dashboard for your homelab servers**

[Features](#features) • [Quick Start](#quick-start) • [Screenshots](#screenshots) • [Architecture](#architecture)

</div>

---

## ✨ Features

- 🖥️ **Multi-server monitoring** - Monitor unlimited Linux/Windows/macOS machines
- 📊 **Real-time metrics** - CPU, Memory, Disk, GPU usage with 10s refresh
- 🔒 **SSH-based** - No exposed ports, works through SSH tunnels
- 🎨 **Beautiful UI** - Binance-inspired design, responsive and modern
- 🪶 **Lightweight** - Single Go binary (~10MB), minimal resource usage
- 🔌 **Port monitoring** - View all TCP/UDP ports on each server
- 🚀 **Easy deployment** - 5-minute setup with example configs

## 🎯 Why This Project?

Existing monitoring solutions are either:
- **Too heavy** (Prometheus + Grafana)
- **Require open ports** (Netdata, Cockpit)
- **Limited features** (Uptime Kuma)

**Homelab Monitor** is designed for homelab enthusiasts who want:
- Centralized monitoring without complex setup
- Security-first (SSH-only, no exposed services)
- Beautiful interface that doesn't look like enterprise software

## 📸 Screenshots

### Dashboard Overview
![Dashboard](https://via.placeholder.com/800x400.png?text=Dashboard+Screenshot)

### Port Monitoring
![Ports](https://via.placeholder.com/800x400.png?text=Port+Monitor+Screenshot)

## 🚀 Quick Start

### Prerequisites

- Go 1.23+ (for building)
- SSH access to your servers
- SSH key authentication configured

### Installation

#### 1. Clone the repository

```bash
git clone https://github.com/Cass-ette/homelab-monitor.git
cd homelab-monitor
```

#### 2. Configure servers

```bash
cp config.example.yaml config.yaml
# Edit config.yaml with your server details
```

#### 3. Build and run

```bash
# Build the monitor server
go build -o monitor main.go

# Build the agent (optional, for Mac/standalone servers)
cd agent && go build -o agent agent.go

# Run the monitor
./monitor
```

#### 4. Access the dashboard

Open http://localhost:8090 in your browser.

## 📋 Configuration

### config.yaml

```yaml
ssh_key: ~/.ssh/id_ed25519

servers:
  production:
    host: prod.example.com
    port: 22
    user: root
    platform: linux

  windows-pc:
    host: 192.168.1.100
    port: 22
    user: administrator
    platform: windows
```

### Supported Platforms

- **Linux**: Direct SSH monitoring
- **Windows**: OpenSSH Server required
- **macOS**: Use the lightweight agent (reports via HTTP)

## 🏗️ Architecture

```
┌─────────────┐
│   Browser   │
└──────┬──────┘
       │ HTTP :8090
       ▼
┌─────────────────┐
│  Monitor Server │ (Go)
│   + Web UI      │
└────┬───────┬────┘
     │       │
     │ SSH   │ SSH
     ▼       ▼
  ┌────┐  ┌────┐
  │ S1 │  │ S2 │  Linux/Windows servers
  └────┘  └────┘
     
     HTTP Report
        ▲
        │
      ┌────┐
      │ S3 │  macOS agent (optional)
      └────┘
```

### Two Monitoring Modes

1. **SSH Mode** (Linux/Windows):
   - Monitor server queries via SSH
   - No agent installation needed
   - Works through reverse tunnels

2. **Agent Mode** (macOS/Standalone):
   - Lightweight agent reports metrics via HTTP
   - Useful when SSH is not available
   - 9MB binary, <0.1% CPU usage

## 🔧 Advanced Setup

### SSH Reverse Tunnels (for home servers behind NAT)

```bash
# On home server, establish reverse tunnel to VPS
ssh -fNR 2222:localhost:22 user@vps.example.com

# Configure monitor to access via tunnel
servers:
  home-server:
    host: localhost
    port: 2222  # tunnel port on VPS
    user: root
```

### macOS Agent Setup

```bash
# Build agent
cd agent && go build -o agent agent.go

# Run agent (reports to monitor at :8090)
./agent

# Or install as LaunchAgent (auto-start on boot)
# See agent/README.md
```

## 🎨 UI Themes

Currently supports **Binance theme** (professional trading UI style). More themes coming soon:
- Web3/Crypto style
- Minimal monochrome
- Custom themes (PR welcome!)

## 📊 Metrics Collected

| Metric | Linux | Windows | macOS |
|--------|-------|---------|-------|
| CPU Usage | ✅ | ✅ | ✅ |
| Memory Usage | ✅ | ✅ | ✅ |
| Disk Usage | ✅ | ✅ | ✅ |
| GPU Usage | ✅ (NVIDIA) | ✅ (NVIDIA) | ❌ |
| Uptime | ✅ | ✅ | ✅ |
| Network Ports | ✅ | ✅ | ⚠️ |

## 🛡️ Security Notes

- **SSH keys only** - No password authentication
- **No exposed ports** - All communication via SSH or local HTTP
- **Read-only queries** - Monitor doesn't modify server state
- **Private by default** - Dashboard only accessible on localhost (use SSH port forwarding for remote access)

### Remote Access (Secure)

```bash
# From your laptop, forward to VPS
ssh -L 8090:localhost:8090 user@vps.example.com

# Then open http://localhost:8090 on your laptop
```

## 🤝 Contributing

Contributions welcome! Please:
1. Fork the repository
2. Create a feature branch
3. Submit a pull request

### Ideas for Contribution

- [ ] Docker container monitoring
- [ ] Alert/notification system
- [ ] Historical data storage (SQLite)
- [ ] More UI themes
- [ ] Custom metric plugins
- [ ] Mobile app

## 📝 License

MIT License - see [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- UI inspired by [Binance](https://www.binance.com)
- Built with [Gin](https://github.com/gin-gonic/gin) and [Tailwind CSS](https://tailwindcss.com)

## 📧 Contact

- GitHub: [@Cass-ette](https://github.com/Cass-ette)
- Issues: [GitHub Issues](https://github.com/Cass-ette/homelab-monitor/issues)

---

<div align="center">
Made with ❤️ for the homelab community
</div>
