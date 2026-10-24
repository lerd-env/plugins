# Submitting a plugin

A plugin runs as the person who installs it, with their permissions, on the machine that holds their code and their databases. These terms exist so that installing anything listed on [plugins.lerd.sh](https://plugins.lerd.sh) is a safe decision. A plugin is approved only when it meets every one of them, and stays listed only while it keeps meeting them.

## How approval works

You open a pull request adding your plugin's entry: its repository and the full 40-character commit you want listed. A maintainer reads the plugin at that commit and approves it or explains what stands in the way. Lerd installs that commit and nothing else, so every update is a new pull request with a new commit, and the maintainer reviews the difference from the last approved one. The pull request opens with a [template](.github/pull_request_template.md): fill in every section, since it is the first thing the maintainer reads.

The entry is `plugins/<name>.yaml`, named like the plugin:

```yaml
repo: https://github.com/acme/queue-board
commit: 0123456789abcdef0123456789abcdef01234567
author: acme
description: Every site's queued and failed jobs on one dashboard page
category: Debugging
tags: [queues, dashboard]
images: [docs/board.png]   # paths in your repository at that commit, the first one is the preview
```

Images stay in your repository, and the catalog copies them from the approved commit: the first is the preview on plugins.lerd.sh and in Lerd's Browse tab. Show them in your pull request's description too, linked at that commit (`https://raw.githubusercontent.com/<owner>/<repo>/<commit>/<path>`), so the reviewer sees what will be listed. [lerd-env/plugins#1](https://github.com/lerd-env/plugins/pull/1) is an example submission.

A check runs on every pull request. It fetches your commit, installs it with the newest Lerd that has plugins, a beta or a build of the coming release when that is newer than the last stable one, beside every plugin already listed, and blocks on what it can verify: a public repository, a license, a README, the `lerd` bound, known `platforms`, no compiled binaries, pinned images your README names, a screenshot when the plugin adds a page, site tab, Overview card or page script, images within their limits, and names nobody else has. It also points the reviewer at lines worth reading, such as minified files, downloads piped to a shell and requests for privilege. A pull request changes only its own entry.

Working with an AI assistant, ask it to use this repository's `submit-plugin` skill: it reviews the plugin with the same `review-plugin` skill maintainers use, then writes the entry and the pull request.

Approval means a maintainer read the code at that commit and found nothing against these terms. It is not a warranty, and the plugin stays your work and your responsibility.

## The repository

1. The repository is public and stays public while the plugin is listed.
2. It carries an open source license that lets anyone read, run and audit the code.
3. Its README says what the plugin does, which parts of Lerd it adds to (commands, hooks, MCP tools, pages, page scripts, notifications, services, frameworks), what it reads, what it changes, and anything it sends over the network.
4. `plugin.yaml` declares `lerd` with the oldest Lerd release the plugin runs on, as a lower bound only. When it only works on Linux or only on macOS, it says so with `platforms`; Windows runs Lerd in WSL, so `linux` covers it.

## Code that can be read

5. Everything the plugin runs is in the approved commit, as source a reviewer can read: scripts in shell, Python, Node or similar. Not PHP: Lerd runs PHP in its containers, not on the machine where plugins run. No compiled binaries, since nobody can review one, and no obfuscated or minified code without the source it was built from. Images and fonts for plugin pages are fine.
6. The plugin never downloads code and runs it. No `curl | sh`, no self-updater, no fetching scripts or modules at run time. Updates reach users only through the catalog.
7. Pages and page scripts bundle their own assets. No scripts, styles or fonts loaded from third-party servers.

## Your users' machine

8. The plugin never asks for privilege: no `sudo`, `pkexec`, setuid helpers, or changes to system files and services.
9. It writes only inside its own folder, the data folder Lerd gives it, and the sites it is run against when that is the plugin's stated purpose. It changes Lerd's state through Lerd's API with the token Lerd gives it, never by editing Lerd's configuration files. When Lerd lacks something the plugin needs, it gets added to Lerd as described in [Extending the Plugin System](https://lerd.sh/contributing/plugins) rather than worked around.
10. It does not read secrets it has no stated need for: SSH and GPG keys, browser profiles, credential stores, other applications' data. A site's `.env` is read only when the README says why.
11. The token Lerd hands it never leaves the machine, and is never written to logs or files outside the plugin's data folder.
12. Hooks finish quickly and fail safely. A failing hook must never leave a site, a service or a database half changed.

## Data and the network

13. Nothing about the user, their projects or their machine leaves it without an explicit opt-in the user turns on themselves, and the README names what is sent and to whom.
14. No analytics, tracking, advertising or crypto mining, in the plugin or in anything it serves. A plugin that sends notifications declares `notifications` in its `plugin.yaml`, which Lerd asks the user to allow, and sends one only for something the user needs to know or act on, never progress or promotion.
15. A page script runs inside the user's own sites, in their origin. It may only talk to its own plugin endpoints, and never reads or sends the page's content, cookies or storage anywhere else.

## Services and frameworks

16. Every service names its source in the README: the image it runs, the registry it comes from, and the project that publishes it, with a link.
17. Images come from a public registry and are pinned to a version tag, preferably a digest. Never `latest`.
18. A service or framework never reuses a name Lerd, its stores or another listed plugin already has. Lerd refuses those at install anyway.

## Honesty

19. The name, description and images describe what the plugin actually does. No impersonating Lerd, another plugin or another author, and no claim to be official.
20. Images you submit are yours to publish, and show the plugin itself. A plugin that adds a page, a site tab, an Overview card or a page script lists at least one. Up to 6 images, each a PNG, JPEG or WebP under 1 MB and at most 2560 px wide; a 16:10 screenshot fills the card best, and the first one is the preview.

## Staying listed

21. Security problems reported to you are acknowledged within 7 days and fixed or answered within 30. Report a problem in someone else's plugin privately with **Report a vulnerability** on this repository's Security tab, not in a public issue.
22. Maintainers may delist a plugin at any time: a vulnerability, a breach of these terms, an abandoned repository, or a commit history rewritten under an approved commit. Delisting stops updates and removes the plugin from Browse. It never uninstalls anything from a user's machine.
23. These terms may change. A listed plugin gets a reasonable window to meet a new term before it affects the listing, unless the term closes a risk to users that is already being exploited.
