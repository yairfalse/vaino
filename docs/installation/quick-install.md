# Quick Install VAINO - No BS, Just Works

## For Users: Install Right Now

### Option 1: Quick Install (Recommended)
```bash
curl -sSL https://raw.githubusercontent.com/yairfalse/vaino/main/scripts/simple-install.sh | bash
```

### Option 2: Build from Source
```bash
git clone https://github.com/yairfalse/vaino.git
cd vaino
make install
```

### Option 3: Manual Build
```bash
go build -o vaino ./cmd/vaino
sudo mv vaino /usr/local/bin/
```

## For Maintainers: Create a Release

### Easy Way:
```bash
make release VERSION=0.1.0
```

Then:
1. Go to https://github.com/yairfalse/vaino/releases/new
2. Create tag `v0.1.0`
3. Upload files from `dist/`
4. Publish

### Manual Way:
```bash
./scripts/manual-release.sh 0.1.0
```

## That's It!

No complex CI, no Docker Hub secrets, no Homebrew tokens. Just working software.

Once you have a release up, the installer will use binaries. Until then, it builds from source.

```bash
vaino scan --provider terraform
vaino diff
```