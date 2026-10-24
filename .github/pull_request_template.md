<!--
Thanks for submitting a plugin. Read SUBMITTING.md first: a pull request adds or
updates one entry, plugins/<name>.yaml, pinned to a full 40-character commit.
Fill in every section; "nothing" is a fine answer where it is true.
-->

## Plugin

- Repository: https://github.com/<owner>/<repo>
- Commit: <!-- the full 40-character commit Lerd will install -->
- Previously approved commit: <!-- updates only; the reviewer reads the difference from it -->

## What it does

<!-- A few sentences for someone deciding whether to install it. -->

## What it adds to Lerd

<!-- Commands, site commands, hooks, MCP tools, API endpoints, a dashboard page, site tabs, Overview cards, page scripts, notifications, services, frameworks, jobs it takes over under `provides`. -->

## What it reads, changes and sends

- Reads: <!-- Lerd's API, files, a site's .env (and why) -->
- Changes: <!-- files it writes, settings, services; "nothing" if so -->
- Sends over the network: <!-- what, to whom, and the opt-in that turns it on; "nothing" if so -->

## Requirements

<!-- Commands it calls beyond its interpreters (requires), the oldest Lerd it runs on (lerd), platforms if not both. -->

## Images

<!-- Required for a page, site tab, Overview card or page script. Link each at the commit:
![What it shows](https://raw.githubusercontent.com/<owner>/<repo>/<commit>/<path>) -->

## Checklist

- [ ] The repository is public and has an open source license.
- [ ] The README says what the plugin adds, reads, changes and sends.
- [ ] `plugin.yaml` declares `lerd` as a lower bound, and `platforms` if it runs on only one.
- [ ] Everything it runs is readable source in the commit, in shell, Python, Node or similar but not PHP: no binaries, no minified code without its source, nothing downloaded and run.
- [ ] It asks for no privilege and writes only to its own folder, its data folder and the sites it is run against, changing Lerd only through its API.
- [ ] It reads no secrets it has no stated need for, and a site's `.env` only when the README says why.
- [ ] Lerd's token never leaves the machine and is never written outside the plugin's data folder.
- [ ] Hooks finish quickly and fail without leaving a site, service or database half changed.
- [ ] Nothing leaves the machine without an opt-in the user turns on, and there are no analytics, tracking, ads or mining.
- [ ] Any notifications are declared in `plugin.yaml` and only tell the user something they need to know or act on.
- [ ] Pages bundle their own assets, nothing loads from third-party servers, and page scripts talk only to the plugin's own endpoints.
- [ ] Any service image comes from a public registry, is pinned to a version tag or a digest, never `latest`, and is named in the README with its source.
- [ ] No service or framework reuses a name Lerd, its stores or another plugin already has.
- [ ] The name, description and images describe what it actually does.
- [ ] The images show the plugin itself, are mine to publish, and are at most 6, each a PNG, JPEG or WebP under 1 MB and at most 2560 px wide.
