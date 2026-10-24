<h1 align="center">Lerd Plugins</h1>

<p align="center">
  The registry of approved plugins for <a href="https://lerd.sh">Lerd</a>.<br>
  Extend Lerd with your own commands, dashboard pages, AI tools and services, no fork and no binary release required.
</p>

<p align="center">
  <a href="https://lerd.sh"><img src="https://img.shields.io/badge/part%20of-lerd-ff2d20" alt="Part of Lerd"></a>
  <a href="https://lerd.sh/features/plugins"><img src="https://img.shields.io/badge/docs-lerd.sh-blue" alt="Docs"></a>
  <a href="https://plugins.lerd.sh"><img src="https://img.shields.io/badge/browse-plugins.lerd.sh-ff2d20" alt="Browse plugins"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-MIT-blue.svg" alt="License: MIT"></a>
</p>

Open **System > Plugins > Browse** in the Lerd dashboard, or run `lerd plugin install <name>`, and Lerd installs a plugin from this registry: a folder of scripts and a `plugin.yaml` that tells Lerd where they plug in. A plugin can add a command to your terminal, a page to the dashboard, a tool to your AI assistant, a service to the picker, or take over a job Lerd does, all without touching Lerd itself.

That's the whole point: **Lerd stays small, and everything your team needs on top of it lives here.** A queue board for every site, a debug bar on your pages, a copy-on-write database branch per worktree, a deploy button wired to your hosting, written once as a plugin and installable by everyone.

And because a plugin runs as you, every listing here was **read by a maintainer at one exact commit, and that commit is the only one Lerd installs.** A repository that changes after approval changes nothing on your machine.

## What a plugin can do

A single `plugin.yaml` and the scripts it names can give Lerd:

- ⌨️ **CLI commands** — `lerd <plugin>:<command>`, with the site you're standing in handed over
- 🧭 **Site commands** — in every site's Commands menu and the command palette, streamed live to the dashboard
- 🪝 **Hooks** — run on ten events, from a site being linked to a service starting
- 🤖 **MCP tools and a skill** — new tools for your AI assistant, with instructions `lerd mcp:inject` writes beside Lerd's own
- 🔌 **API endpoints** — under `/api/plugins/<name>/`, behind the same access gate as Lerd's
- 🖥️ **Dashboard pages and site tabs** — a page on the rail and a tab on every site, in a sandboxed frame
- 🌐 **Scripts on your sites' pages** — injected into the sites you switch the plugin on for
- 📦 **Services and frameworks** — presets in the service picker and framework definitions, written like the stores'
- 🔁 **Jobs Lerd hands over** — like creating every worktree's database, so a copy-on-write tool replaces a full dump and import

Scripts are plain shell, Python or Node. Lerd checks what each one needs before installing it, gives every surface a time limit, and when something fails it offers a filled-in issue on the plugin's own repository.

## Available plugins

The first plugins are being reviewed. Browse what's listed at [plugins.lerd.sh](https://plugins.lerd.sh), or be the first: see [Contributing](#contributing).

## Usage

```bash
lerd plugin search [term]      # browse the approved plugins
lerd plugin install <name>     # install a listed plugin at its approved commit
lerd plugin list               # what's installed, and where each came from
lerd plugin update [name...]   # move to newer approved commits, keeping each plugin's data
lerd plugin remove <name>      # remove it, keeping its data folder
lerd plugin new <name>         # scaffold your own from a working boilerplate
```

Lerd reads this registry at most once a day and keeps the last copy, so browsing and installed plugins work offline. A plugin states the oldest Lerd it runs on and, when it matters, whether it runs on Linux or macOS (Windows runs Lerd in WSL, so Linux covers it). Browse shows that, and what it requires on your machine, before you install it.

## Contributing

New plugins are welcome, and so is anything that makes reviewing them easier.

1. Write your plugin in a public repository with an open source license, starting from `lerd plugin new <name>` and the [plugin guide](https://lerd.sh/features/plugins)
2. Read the [approval terms](SUBMITTING.md)
3. Fork this repo and add `plugins/<name>.yaml`
4. Open a pull request

### Entry schema

```yaml
repo: https://github.com/acme/queue-board
commit: 0123456789abcdef0123456789abcdef01234567   # the full commit to approve
author: acme
description: Every site's queued and failed jobs on one dashboard page
category: Debugging
tags: [queues, dashboard]
images: [docs/board.png]   # paths in your repository at that commit, the first one is the preview
```

A plugin pulled for a security problem keeps its entry, marked revoked, so Lerd can tell everyone who has it:

```yaml
revoked:
  reason: Sends the site's .env to a third party   # shown to every user who has it
  at: "2026-10-24"
  commits: [0123456789abcdef0123456789abcdef01234567]   # the commits it covers; all of them when left out
```

With `commits` listed, a fixed commit can be approved again in the same entry while the ones named stay revoked. A plugin that is only delisted has its entry deleted, and Lerd leaves what is installed alone.

A check runs on every pull request. It fetches your commit, installs it with the newest Lerd that has plugins, a beta or a build of the coming release when that is newer than the last stable one, beside every plugin already listed, and blocks on what it can verify: a public repository, a license, a README, the `lerd` bound, known `platforms`, no compiled binaries, pinned images your README names, a screenshot when the plugin adds a page, site tab, Overview card or page script, images within their limits, and names nobody else has. It points the reviewer at anything else worth reading, then a maintainer reads the code. An update is a new pull request with the new commit.

If your plugin needs something Lerd does not offer yet, add it to Lerd rather than working around it: see [Extending the Plugin System](https://lerd.sh/contributing/plugins).

## Reporting a problem

Plugins are written by their authors, not by Lerd. A listing means a maintainer read the code at one commit; it is not a warranty or a security audit, and Lerd and its maintainers are not responsible for what an installed plugin does. Install only plugins you trust.

A bug in a plugin belongs on that plugin's repository, and Lerd's dashboard links straight to a filled-in issue there when a plugin fails. A security problem in a listed plugin goes privately through **Report a vulnerability** on this repository's Security tab.

## License

MIT
