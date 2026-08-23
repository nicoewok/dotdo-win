# DOT ● DO — Android App Design & Style Guide

This style guide breaks down the design, colors, UI layout, fonts, and data structure of **DOT ● DO** so you can easily build a matching Android application.

---

## 1. Overview & Concept

**DOT ● DO** is a dark-themed, minimalist task manager. It focuses on simplicity, clear visual status indicators, and retro pixel typography.

Key features for the Android app:
- **Simple Status Cycle**: Tapping a task toggles its state (`todo` ➔ `doing` ➔ `done` ➔ `todo`).
- **Dark Retro Theme**: Pure dark background with glowing red, orange, and green status accents.
- **Bunny Mascot**: Cute pixel bunny ASCII art displayed at the top of the main screen.

---

## 2. Bunny ASCII Art

Display this exact ASCII art in the header area using a monospace pixel font (`DotGothic16` or standard Android monospace `monospace`).

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
```

---

## 3. Color Palette

Use these exact colors for your Android app theme (e.g. in `colors.xml` or Jetpack Compose `Color` values):

| Color Name | Hex Code | RGB Values | Simple Description & Usage |
| :--- | :--- | :--- | :--- |
| **Dark Background** | `#0E0E12` | `rgb(14, 14, 18)` | Main screen background behind everything. |
| **Card Background** | `#18181E` | `rgb(24, 24, 30)` | Background for task cards and top header container. |
| **Input Background** | `#23232D` | `rgb(35, 35, 45)` | Background for text inputs and secondary buttons. |
| **Accent Red** | `#EB4B4B` | `rgb(235, 75, 75)` | App logo dot (`●`), `+ Add` primary button, `todo` status indicator. |
| **Accent Orange** | `#FFAA32` | `rgb(255, 170, 50)` | `doing` status indicator (`◐`). |
| **Accent Green** | `#55D782` | `rgb(85, 215, 130)` | `done` status indicator (`✔`). |
| **Text Primary** | `#F5F5FA` | `rgb(245, 245, 250)` | High-contrast white for task titles and main text. |
| **Text Muted** | `#A0A0AF` | `rgb(160, 160, 175)` | Soft grey for dates, subtitles, and secondary labels. |

---

## 4. Typography & Fonts

- **Primary App Font**: `scientifica` (slender pixel font) or any crisp pixel/monospace font.
- **Mascot Font**: `DotGothic16` or Android standard `monospace`.

### Text Sizes
- **Header Title**: `30sp` bold (`DOT ● DO`)
- **Main Buttons & Task Titles**: `18sp`
- **Subtitles & Badges**: `16sp`
- **Helper Labels & Dates**: `14sp`

---

## 5. UI Layout & Components

### A. Top Header Banner
- **Left**: Bunny ASCII art (`#F5F5FA` white).
- **Middle**:
  - App Name: `DOT` (White) `●` (Red Accent `#EB4B4B`) `DO` (White).
  - Status Subtitle: `● Synced` in Muted Grey (`#A0A0AF`).
- **Right**: Primary `+ Add` button in Red Accent (`#EB4B4B`).

### B. Task Card Item
Each task is wrapped in a rounded card container (`#18181E` background):

1. **Status Icon Button (Left side)**:
   - **`todo`**: Displays `●` in Red (`#EB4B4B`).
   - **`doing`**: Displays `◐` in Orange (`#FFAA32`).
   - **`done`**: Displays `✔` in Green (`#55D782`).
   - *Behavior*: Tapping cycles through `todo` ➔ `doing` ➔ `done` ➔ `todo`.
2. **Task Title (Middle)**:
   - Normal text (`#F5F5FA`) when `todo` or `doing`.
   - Strikethrough text with muted color (`#A0A0AF`) when `done`.
3. **Due Date (Below Title)**:
   - Optional formatted date (e.g. `2026-08-25`) in Muted Grey (`#A0A0AF`).
4. **Delete Button (Right side)**:
   - `✕` button to delete the task.

### C. Bottom Control Bar
- **Task Summary**: Displays counts e.g., `2 pending, 1 done`.
- **Toggle Done Tasks Button**: Switches between showing all tasks or hiding completed tasks.
- **Purge Done Button**: Deletes all completed (`done`) tasks at once.

### D. Add Task Screen / Dialog
- **Header**: `Add New Task` title with a `Back` button.
- **Title Input Field**: Single-line text input (`#23232D` background) for task name.
- **Due Date Input Field**: Optional date picker or text input (`YYYY-MM-DD`).
- **Buttons**:
  - `Submit` (Red Accent `#EB4B4B` background)
  - `Cancel` (Input background `#23232D`)

---

## 6. Data Format (`tasks.json`)

Tasks are stored in a simple JSON structure.

### Example JSON
```json
{
  "tasks": [
    {
      "id": 1,
      "title": "Buy groceries",
      "status": "todo",
      "due": "2026-08-25T00:00:00Z"
    },
    {
      "id": 2,
      "title": "Build Android app",
      "status": "doing",
      "due": "0001-01-01T00:00:00Z"
    },
    {
      "id": 3,
      "title": "Setup repository",
      "status": "done",
      "due": "0001-01-01T00:00:00Z"
    }
  ]
}
```

### JSON Fields Breakdown

| Field Name | Type | Required? | Simple Explanation |
| :--- | :--- | :--- | :--- |
| **`tasks`** | Array | **Required** | The main list holding all task objects. |
| **`id`** | Number (Integer) | **Required** | A unique number identifying each task (e.g., `1`, `2`, `3`). |
| **`title`** | String | **Required** | The text description of the task (e.g., `"Buy groceries"`). Cannot be empty. |
| **`status`** | String | **Required** | The current state. Must be one of three values: `"todo"`, `"doing"`, or `"done"`. |
| **`due`** | String | *Optional* | The due date in ISO 8601 format (`YYYY-MM-DDTHH:MM:SSZ`). If no due date is set, it uses `"0001-01-01T00:00:00Z"` or can be omitted. |

