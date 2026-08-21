# dotdo-win

> Desktop task manager built with Go and Gio (`gioui.org`).

`dotdo-win` is a native desktop todo application built with Go and the **Gio** immediate-mode GUI framework (`gioui.org`). It displays your task list and keeps your tasks synchronized across devices via Git.

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

## Features & Advantages of Gio

- **Pure Go Desktop GUI**: Built using Gio (`gioui.org`), which compiles directly into native Windows Direct3D 11 graphics.
- **No CGO Required**: Runs cleanly on Windows with `CGO_ENABLED=0` or standard Go toolchain without requiring external MinGW GCC setups.
- **Task Management**: Automatically loads local JSON tasks (`%USERPROFILE%\.dotdo\tasks.json`) and displays task statuses (`●` todo, `◐` doing, `✔` done).

---

## Building and Running

Simply run with the standard Go toolchain:

```cmd
# Run directly:
go run .

# Or build the executable:
go build -o dotdo.exe .
```

---

## Project Structure

- `main.go`: Gio GUI application entry point.
- `pkg/service`: High-level task operations (add, list, status, delete, git sync).
- `pkg/store`: Local JSON storage manager (`%USERPROFILE%\.dotdo\tasks.json`).
- `pkg/task`: Task data model.
- `docs/THEME_AND_GUI.md`: Reference documentation for the original Fyne Monochrome Theme.
