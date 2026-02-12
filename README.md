# BaoBun
BaoBun is a headless BaoSwarm client with an embedded web UI.

## Quickstart (Auto Setup + Build)
Run one command from the repo root (`BaoBun/`).

### Windows
```powershell
.\auto\setup.ps1
```
or
```cmd
auto\setup.cmd
```

### Linux
```bash
bash ./auto/setup.sh
```

### macOS
```bash
bash ./auto/setup.sh
```

What the auto scripts do:
1. Install missing prerequisites (Go, Node.js, npm).
2. Build frontend assets (`internal/webui/dist`).
3. Compile binaries into `bin/`.

Output binaries:
- `bin/baobun-client` (`bin/baobun-client.exe` on Windows)
- `bin/baobun-maker` (`bin/baobun-maker.exe` on Windows)

Run after build:
- Windows:
```powershell
.\bin\baobun-client.exe
```
- Linux/macOS:
```bash
./bin/baobun-client
```

Current client mode starts local web endpoints:
- `http://localhost:8880`
- `http://localhost:8881`
- `http://localhost:8882`
- `http://localhost:8888`

### Seed Configuration (Frontend)
- Open any UI endpoint and click `Config`.
- Enter your own 4 seeds and save, or click `Auto Generate + Save`.
- Each seed must be exactly `32` characters.
- Saved seeds are persisted to `seeds.json` at repo root.
- Restart `baobun-client` after saving to apply new seeds.

### Proof Cache Persistence
- Validated transfer proofs are persisted on disk per swarm.
- Cache location: `<download_dir>/.baobun/proofs/<infohash>/`.
- After restart, partial clients can continue serving units they can prove.
- For legacy partial data without cached proofs, those units are not advertised for upload until the node has a proof (or completes the full file).

### Drag And Drop Import
- You can drag and drop one or more files anywhere on the UI.
- `.bao` files are imported as metadata.
- Non-`.bao` files are accepted and converted into new swarms automatically.
- Bao details `Files` tab now shows file path, size, remaining bytes, and progress.

## Manual Setup
Use this if you do not want the auto scripts.

### Requirements
- Go `1.25+`
- Node.js `22.12+` and npm
- Internet access for module/package downloads

### Install Prerequisites

#### Windows (PowerShell)
```powershell
winget install -e --id GoLang.Go
winget install -e --id OpenJS.NodeJS.LTS
```

#### Linux (Ubuntu/Debian)
```bash
sudo apt-get update
sudo apt-get install -y ca-certificates curl git golang-go nodejs npm
```

#### Linux (Fedora/RHEL/CentOS)
```bash
sudo dnf install -y ca-certificates curl git golang nodejs npm
```

#### Linux (Arch)
```bash
sudo pacman -Sy --noconfirm ca-certificates curl git go nodejs npm
```

#### macOS
```bash
brew install go node
```

### Build Manually
From the repo root:

```bash
cd internal/webui
npm ci
npm run build
cd ../..
mkdir -p bin
go mod download
go build -o ./bin/baobun-client ./cmd/client
go build -o ./bin/baobun-maker ./cmd/maker
```

Windows PowerShell equivalent:
```powershell
Set-Location .\internal\webui
npm ci
npm run build
Set-Location ..\..
New-Item -ItemType Directory -Force -Path .\bin | Out-Null
go mod download
go build -o .\bin\baobun-client.exe .\cmd\client
go build -o .\bin\baobun-maker.exe .\cmd\maker
```

### Run Manually
- Windows:
```powershell
.\bin\baobun-client.exe
```
- Linux/macOS:
```bash
./bin/baobun-client
```
