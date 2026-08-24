# dotdo (Windows)

> Simple todo-list app with syncing

`dotdo-win` is a native desktop todo application built with Go. It displays your task list and keeps your tasks synchronized across devices via Git.

The original dotdo is a CLI/TUI tool that has the same functionality and art style. If you use Linux please try [this project](https://github.com/nicoewok/dotdo) instead!
If you use Android then try using the [android app](https://github.com/nicoewok/dotdo-android)

---

## Bunny
```text
    ⠏⢣ ⠏⢣
  ⢠⡶⠧⠧⠶⠧⠧⠶⢶⡄
  ⡜         ⢣
 ⢸   ⠛   ⠛  ⢣
  ⢣      Y  ⢸
  ⢸      "  ⡜
  ⡜        ⢸
⠺⡜         ⡜
  ⠙⠒⠤⣀⣀⣇⣸⣇⣸
 DOT ● DO bunny  
```

---

## Installation & Features

### Installation & Shortcuts
- Download and run the `dotdo-Setup-1.0.0.exe` installer.
- Interactive setup options allow you to automatically add dotdo to your **Start Menu** and create a **Desktop** shortcut.
- All application assets (fonts, icons) are embedded directly into `dotdo.exe`, making the app fully portable anywhere on Windows.

### Task Synchronization via GitHub
- To synchronize your tasks across devices, you need a **private GitHub repository** named `.dotdo` (e.g. `your-username/.dotdo`).
- Within dotdo, click **Connect GitHub** and enter your Personal Access Token (PAT) with `repo` scope.
- dotdo syncs your `tasks.json` with your private `.dotdo` repository when you click **Pull** or **Push**.
- For detailed technical documentation on link generation, config structure, and token storage, see [docs/GITHUB_INTEGRATION.md](file:///d:/dev/dotdo-win/docs/GITHUB_INTEGRATION.md).

### Windows Uninstaller
Uninstall via "Add or remove programs". Removes all program files, Start Menu/Desktop shortcuts, and completely deletes local user data folders (`%APPDATA%\dotdo` containing `tasks.json` & `config.json` and `%LOCALAPPDATA%\dotdo`).

---

## Building from Source & Packaging Installer

### Requirements
- **Go 1.22+**
- **Git**
- **Inno Setup 6** (optional, required only for compiling the `dotdo-Setup-1.0.0.exe` installer)

### Building the Executable
```powershell
go build -ldflags="-H=windowsgui" -o dotdo.exe .
```

### Packaging & Generating Installer
To compile the GUI executable, package assets into a portable ZIP release, and build the Windows setup installer:

```powershell
powershell -ExecutionPolicy Bypass -File .\installer\package.ps1
```

This generates the following artifacts in the `dist/` directory:
- `dist/dotdo-v1.0.0-windows-amd64.zip` (Standalone portable package)
- `dist/dotdo-Setup-1.0.0.exe` (Windows Setup Installer, compiled via Inno Setup `ISCC.exe`)

---

## Credits

- **Font**: [scientifica](https://github.com/nerdypepper/scientifica) by **NerdyPepper** — a tall, slender pixel font for coders.
