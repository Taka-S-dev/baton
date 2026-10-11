# baton

[![CI](https://github.com/Taka-S-dev/baton/actions/workflows/ci.yml/badge.svg)](https://github.com/Taka-S-dev/baton/actions/workflows/ci.yml)
[![Release](https://github.com/Taka-S-dev/baton/actions/workflows/release.yml/badge.svg)](https://github.com/Taka-S-dev/baton/actions/workflows/release.yml)

```
  _           _
 | |__   __ _| |_ ___  _ __
 | '_ \ / _` | __/ _ \| '_ \
 | |_) | (_| | || (_) | | | |
 |_.__/ \__,_|\__\___/|_| |_|
```

A terminal-based workflow runner for Windows, Linux, and macOS. Define commands in a config file, select and execute them interactively.

## Why baton?

Honestly: laziness. I didn't want to memorize each project's command combinations. I didn't want to babysit the terminal, typing the next command each time one finished. And I wanted a new command to be a near-copy of an existing one — pick a slotted command as a template, fill in different values, done. baton is those habits turned into a tool.

Task runners are a crowded space — [just](https://github.com/casey/just), [go-task](https://github.com/go-task/task), or a dozen lines of [fzf](https://github.com/junegunn/fzf) glue cover much of what baton does. baton exists for a combination none of them offers on its own:

- **Windows as a first-class citizen.** The TUI and every feature behave the same in cmd.exe, PowerShell, and Unix shells — no WSL or bash dependency, and each command chooses its shell (`cmd`/`sh` by default, `ps` opt-in).
- **No DSL to learn.** Commands are plain JSON or TSV, and once a project is set up the TUI creates and edits them for you.
- **Interactive placeholders.** `{slots}` resolve at run time from selection lists with a live command preview — closer to a snippet manager like [pet](https://github.com/knqyf263/pet) than to make, but with saved workflows on top.
- **Workflows as data.** Frequent sequences are saved, searched, and re-run from a menu instead of being encoded in a build file.

If you live happily in a Justfile or an fzf pipeline, those remain great choices. baton is for keeping a per-project command palette that anyone can drive from a menu.

Named after the relay baton: each step hands off to the next.

See [docs/reference.md](docs/reference.md) for configuration, screens and keys.

## License

MIT License — see [LICENSE](LICENSE) for details.
