# GitHub Integration Architecture in dotdo

This document explains the architecture and implementation details of GitHub synchronization in **dotdo**, including user setup guidance, browser token link generation, configuration storage, and credential security across platforms.

---

## 1. Connection Screen & Setup Flow

- **Navigation**: In `main.go`, clicking **Connect Github** sets `state.activeView = "github_connect"`, rendering the connection interface.
- **User Instructions**:
  1. Create a **private GitHub repository** named `.dotdo` (e.g. `your-username/.dotdo`).
  2. Click **Generate GitHub Token** to open GitHub's Personal Access Token page.
  3. Ensure the **`repo`** permission scope is checked.
  4. Set expiration to **No expiration** (or custom duration) so re-authentication is not required.
  5. Paste the token (`ghp_...` or `github_pat_...`) into the text field and click **Connect**.

---

## 2. Link Generation & Opening Browser

- **Pre-filled GitHub URL**:
  ```text
  https://github.com/settings/tokens/new?scopes=repo&description=dotdo-windows
  ```
  - `scopes=repo`: Automatically checks the `repo` permission scope checkbox on GitHub.
  - `description=dotdo-windows`: Pre-fills the token note name.
- **Browser Execution**:
  In `main.go` (`openURL` function):
  ```go
  exec.Command("rundll32", "url.dll,FileProtocolHandler", rawURL).Start()
  ```
  *(Fallback: If `rundll32` fails, it falls back to PowerShell `Start-Process '<URL>'`).*

---

## 3. Configuration Storage (`config.json`)

- **Path**: `%LOCALAPPDATA%\dotdo\config.json`
- **Structure**:
  ```json
  {
    "owner": "your-username",
    "repo": ".dotdo",
    "branch": "main"
  }
  ```
- **Security & Scope**: `config.json` contains **only non-sensitive repository settings** (`owner`, `repo`, `branch`). It does **NOT** contain the PAT. `config.json` is explicitly ignored by `.gitignore` so it is never committed or pushed to Git.

---

## 4. PAT Credential Security: Windows vs. Android

### On Windows
- Saved in **Windows Credential Manager** (OS-level encrypted vault) using `github.com/zalando/go-keyring` under service `dotdo` and key `access_token`.
- The token is **never** written to plaintext files or JSON configs on disk.

### For Android Portability
- **Option A: Android KeyStore / EncryptedSharedPreferences (Recommended)**:
  - Android provides **Android KeyStore System** and **EncryptedSharedPreferences** to encrypt secrets at rest using hardware-backed keys.
- **Option B: App-Private Storage (`config.json`)**:
  - On Android, every app runs in its own isolated Linux UID sandbox with private storage (`/data/data/com.dotdo.app/files/`).
  - Storing the PAT in app-private storage with strict `0600` permissions keeps it isolated from other apps on unrooted Android devices.

