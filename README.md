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
