# dotdo (Linux)
> Minimalist, dot-matrix, focused.

`dotdo` is an open-source todo list CLI/TUI tool. It manages your tasks through a monochromatic interface, featuring a pixel Bunny mascot.

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

# Install

Go to the [Releases](https://github.com/nicoewok/dotdo/releases) page and download the latest `.deb` file, then run:
```bash
# Replace x.x.x with the version number
sudo apt install ./dotdo_x.x.x_amd64.deb
```

## Usage

Initialize the local storage and autocomplete:
```bash
dotdo init
```

If you want to sync your data, **dotdo** uses a private Git repository in your home directory to sync tasks across devices automatically.
1. Create a private repository (I named it `.dotdo`)
2. Be sure you have [Git](https://git-scm.com/install/) installed
3. Make sure you are authenticated for Github on your CLI using an [Access Token](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/managing-your-personal-access-tokens#creating-a-fine-grained-personal-access-token)
> To make sure your CLI remembers this token the next time, be sure to execute this before logging in to git:
    ```
    git config --global credential.helper store
    ```

Now follow the instructions displayed during ```dotdo init``` and you are good to use the following commands:

### Commands

- `dotdo` — Show all tasks not on done.

- `dotdo help` — Show all commands.

- `dotdo init` — Initilize the `tasks.json` file to start adding tasks. Add autocomplete to your bash/zsh configuration.

- `dotdo add [task name] -d [date]` — Add a new task. The name does not need to be in "". Example:
```bash
dotdo add Feed bunny
```
You can optionally add a due date using `-d` and adding a date in the **YYYY-MM-DD** format. Example:
```bash
dotdo add Feed bunny -d 2026-01-01
```


- `dotdo doing [task name]` — Mark task as "currently doing". This command has auto-complete for the task name.

- `dotdo done [task name]` — Mark task as "done". This command has auto-complete for the task name.

- `dotdo remove` — Deletes all "done" tasks to clean up `tasks.json`.

- `dotdo sync` — Pulls changes for `tasks.json` & pushes new changes.

- `dotdo list` — Show all tasks. Even "done" ones.



# Build yourself

1. Ensure you have [Go](https://go.dev/doc/install) and [Git](https://git-scm.com/install/) installed
2. ```git clone``` this repository
3. ```go install```
4. Be sure your `go/bin` folder is added to your `PATH`:
    
    a. Get your go folder using
    ```bash
    go env GOPATH
    ```
    b. It should return something like `/home/yourname/go`. If you exedcuted 3. then in this path in `/bin` you should find an executable called `dotdo`
    c. Edit your bash config:
    ```bash
    nano ~/.bashrc
    ```
    d. Go to the last line and paste this line:
    ```bash
    export PATH=$PATH:$GOPATH/bin
    ```
    where `$GOPATH` is the path you got from a. and b. so for example:
    ```bash
    export PATH=$PATH:/home/USERNAME/go/bin
    ```
    e. Save and exit: Ctrl+O -> Enter -> Ctrl+X
    f. Reload your terminal!
