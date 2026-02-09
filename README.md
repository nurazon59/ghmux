# ghmux

CLI for managing multiple GitHub accounts. Automatically switches `gh` auth and git author based on org rules.

## Prerequisites

- [gh](https://cli.github.com/) (GitHub CLI) with multiple accounts logged in

## Install

```bash
go install github.com/nurazon59/ghmux@latest
```

## Config

`~/.config/ghmux/config.yaml`

```yaml
defaults:
  profile: personal

profiles:
  work:
    gh_user: work-login
    git:
      name: Work Name
      email: work@company.com
  personal:
    gh_user: personal-login
    git:
      name: Personal Name
      email: me@gmail.com

rules:
  - hostname: github.com
    org: my-company
    profile: work
```

Rules are evaluated top-down. First match wins. Unmatched orgs fall back to `defaults.profile`.

## Commands

### `ghmux sync`

Switch `gh` auth based on the current directory's git remote. Useful as a `chpwd` hook.

```bash
cd ~/src/github.com/my-company/api
ghmux sync
# → gh auth switch --user work-login
```

### `ghmux init`

Generate per-profile gitconfig files and `includeIf` blocks in `~/.gitconfig`.

```bash
ghmux init
# → generates ~/.gitconfig-work, ~/.gitconfig-personal
# → inserts includeIf block into ~/.gitconfig
```

This ensures `git commit` uses the correct author without any wrapper — matching by remote URL via `hasconfig:remote.*.url`.

## Usage

Run `ghmux sync` automatically on every `cd` by adding a shell hook.

### zsh

Add to `~/.zshrc`:

```zsh
autoload -U add-zsh-hook
_ghmux_chpwd() { ghmux sync 2>/dev/null }
add-zsh-hook chpwd _ghmux_chpwd
```

### fish

Create `~/.config/fish/conf.d/ghmux.fish`:

```fish
function _ghmux_cd --on-variable PWD
    ghmux sync 2>/dev/null
end
```

## License

MIT
