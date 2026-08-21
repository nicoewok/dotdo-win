# dotdo-win

> Monochromatic desktop task manager backed by automated Git sync.

`dotdo-win` is a native desktop todo application built with Go and Fyne (`fyne.io/fyne/v2`). It features a high-contrast dark monochrome pixelated dotty aesthetic (`DOT ● DO`) and keeps your tasks seamlessly synchronized across devices via a private Git repository.

---

## Features

- **Native Desktop GUI**: Built using Fyne v2 with a sleek, dark monochrome pixel-inspired theme.
- **Task Management**: Create tasks, set optional due dates (`YYYY-MM-DD`), mark tasks as active ("Focus") or completed, and delete individual tasks.
- **Automated & Manual Git Sync**: Automatically saves tasks to local JSON and synchronizes changes in the background via Git (`git add`, `commit`, `pull --rebase`, and `push`). Includes a manual **Sync** button and real-time status indicator (`● Synced` / `● Syncing...`).
- **Clean JSON Storage**: Stores all task data in `%USERPROFILE%\.dotdo\tasks.json`.

---

## Data Storage & Git Synchronization Setup

`dotdo-win` stores task data locally in your Windows user home directory under `%USERPROFILE%\.dotdo\tasks.json`.

To sync your task list across devices via Git:

1. Create a private Git repository on GitHub (e.g. named `dotdo-data`).
2. Open PowerShell or Command Prompt and run:
   ```cmd
   cd %USERPROFILE%\.dotdo
   git init
   git remote add origin https://github.com/YOUR_USERNAME/YOUR_PRIVATE_REPO.git
   git branch -M main
   ```
3. Make sure Git authentication is configured on your system (e.g. via GitHub Desktop, Git Credential Manager, or SSH keys).

---

## Building and Running on Windows

### Prerequisites
- [Go 1.20 or newer](https://go.dev/dl/)
- [Git for Windows](https://git-scm.com/download/win)

### Building from Source

Open PowerShell or Command Prompt in the repository folder and execute:

```cmd
go build -o dotdo.exe .
```

To run the application:

```cmd
.\dotdo.exe
```

---

## How to Verify on Windows

To test and verify the installation on your Windows machine:

1. **Launch Application**:
   Run `.\dotdo.exe` from PowerShell or double-click `dotdo.exe`. Confirm the GUI opens with the dark `DOT ● DO` header.

2. **Add Tasks**:
   - Type a task name (e.g. `Setup dotdo sync`) into the title entry field and click **+ Add** (or press Enter).
   - Add another task with an optional due date (e.g. `2026-12-31` in the `Due YYYY-MM-DD` box).

3. **Verify Local JSON Storage**:
   Open `%USERPROFILE%\.dotdo\tasks.json` in Notepad or VS Code to confirm the new tasks are saved properly in JSON format.

4. **Task Interactions**:
   - Click the completion checkbox to mark a task done (verify strikethrough/dimmed state).
   - Click **Focus** to switch a task to `doing` status (symbol changes to `◐`).
   - Click **✖** on a task row to verify individual task deletion.
   - Click **Purge Done** in the footer to clear all completed tasks.

5. **Verify Git Sync**:
   Click the **Sync** button in the top right header. Confirm that the status indicator updates to `● Syncing...` and returns to `● Synced`.
