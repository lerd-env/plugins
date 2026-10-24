---
name: review-plugin
description: Review a Lerd plugin for safety and for Lerd's rules, by reading every file at the pinned commit and then running everything it declares in a throwaway VM before it touches a real machine. Use when asked to check, review or test a plugin pull request, a plugin repository, or your own plugin before submitting it.
---

# Review a plugin

What matters is that approving it is a safe decision: the plugin runs no
malicious code, cannot break Lerd, and follows Lerd's rules. Features working
come second. A plugin runs as the user, with their permissions, so nothing from
it runs on a real machine until it has run clean in a VM.

## The rules it is held to

Read these at the start of every review; they change, so never work from memory.

- `SUBMITTING.md` in this repository: the numbered terms.
- Lerd's plugin guide, the rules plugin authors and their AI assistants are
  given: <https://raw.githubusercontent.com/lerd-env/lerd/main/internal/plugins/guide.md>,
  or `plugin` `guide` through `lerd mcp` when Lerd is installed.
- Lerd's plugin docs, the manifest and every surface:
  <https://raw.githubusercontent.com/lerd-env/lerd/main/docs/features/plugins.md>.

When a question needs Lerd's code, such as what an API endpoint returns or
whether Lerd already offers what the plugin works around, shallow-clone
`lerd-env/lerd` into a temporary folder and read it there.

## 1. Pin what is being approved

- From the PR's `plugins/<name>.yaml` take `repo` and `commit`. Review that
  commit and nothing else: Lerd installs it and only it, so later pushes to the
  repository reach no one until a new update PR is reviewed.
- Clone into a temporary folder and check out the commit. Read the PR's
  comments and the `check` job's output: it lists flagged lines (minified
  files, pipes to a shell, privilege) worth reading first.
- An update must move both: a new `commit` in the entry and a higher `version`
  in the plugin's `plugin.yaml` than the approved commit's. A new commit at the
  same version is sent back, since users and the catalog tell updates apart by
  version. Review the whole diff from the approved commit.

## 2. Read every file at the commit, running nothing

- First find what a plain read misses: submodules (`git submodule status`, code
  from another repository), symlinks (`git ls-files -s`, mode 120000, possibly
  pointing outside the plugin), `.gitattributes` filters or LFS pointers,
  dotfiles, every executable (mode 100755), and every binary that is not an
  image or font. Any of these unexplained is a finding.
- Read every file that runs: `plugin.yaml`, everything under `bin/` or named by
  the manifest, `ui/`, `services/`, `frameworks/`. A script not read is a script
  not reviewed.
- For each surface the manifest declares, write down what it runs and what that
  touches. This list drives step 3.
- Look specifically for:
  - **Network**: `curl`, `wget`, `fetch`, `XMLHttpRequest`, `urllib`,
    `requests`, sockets, `nc`. Anything but `$LERD_API_URL` must be named in the
    README and be opt-in (term 13).
  - **Code from elsewhere**: downloads that run, `eval`, `base64 -d`, built
    strings handed to a shell, `pip` or `npm install` at run time, minified files
    without their source (terms 5, 6).
  - **Privilege**: `sudo`, `pkexec`, `doas`, setuid, writes to `/etc`, system
    services, crontabs (term 8).
  - **Reach**: paths outside `$LERD_PLUGIN_DIR`, `$LERD_PLUGIN_DATA` and the site
    it runs on; reads of `~/.ssh`, `~/.gnupg`, browser profiles, keychains or a
    `.env` the README does not explain (term 10).
  - **Lerd's own state**: writing Lerd's config, data, units, containers or
    binaries is refused (term 9). Reading Lerd's files to fill a gap in its API
    is the workaround term 9 asks for an addition to Lerd instead: name the gap.
  - **The token**: `$LERD_API_TOKEN` logged, written outside the data folder, or
    sent anywhere but `$LERD_API_URL` (term 11).
  - **Page scripts**: reading the page's cookies, storage or content, or talking
    to anything but `/_lerd/plugins/<name>/` (term 15).
  - **Services**: images pinned to a version or digest, from a public registry,
    named in the README with their source (terms 16, 17).
- Hold it to the guide's rules too: `loopback_only` on any endpoint that runs
  an action or changes anything, values from the API or a site set with
  `textContent` or escaped before `innerHTML`, pages built from Lerd's classes
  with no colours of their own, a page script inside a shadow root, portable
  shell when `platforms` includes macos, `requires` naming every command the
  scripts call, and a `lerd:` lower bound no older than the first release with
  the API it uses (plugins exist from 1.38).
- Compare the README's "reads, changes, sends" with what the code does. A gap is
  a finding even when the code is harmless.

Stop with changes requested if anything is malicious or unexplained.

## 3. Run it in a VM

A virtual machine you can throw away, never your own machine. Install the
newest Lerd release the plugin's `lerd:` bound allows, link a real site (a real
framework app, not a stub) and start the services it would touch.

Record Lerd's state, then start watching:

```bash
for s in $(lerd sites | awk 'NR>1{print $2}'); do echo "$s $(curl -sk -o /dev/null -w '%{http_code}' https://$s/)"; done > /tmp/sites-before
md5sum ~/.config/lerd/config.yaml ~/.config/lerd/services/*.yaml > /tmp/config-before
touch /tmp/before
sudo tcpdump -i any -nn 'not port 22 and not host 127.0.0.1' -w /tmp/plugin.pcap &
```

- Install the plugin at the commit: clone it, check the commit out, and
  `lerd plugin install <folder>`.
- Exercise **every** surface it declares: each command and site command with
  and without arguments, each hook by causing its event, each MCP tool through
  `lerd mcp`, each API endpoint with good and bad input, each page, site tab and
  Overview card in a browser over an ssh tunnel, each page script on a site it
  is switched on for, each service installed and started, each framework
  detected, each `provides` job triggered, notifications sent and refused.
- Feed it hostile input wherever it takes any: a missing site, names with
  spaces, quotes, `../`, `$(touch /tmp/pwned)`, markup. An error is fine; a
  write outside its folders, a command that runs, or a change to Lerd is not.
- To tie network traffic to the plugin rather than to Lerd's own store and
  catalog refreshes, trace the processes it runs:
  `sudo strace -f -e trace=connect -p <lerd-ui pid>` while calling its surfaces.
- Remove it with `lerd plugin remove` and confirm nothing of it keeps running.

Then check:

- **Files**: `find ~ /tmp /etc -xdev -newer /tmp/before -type f`. Anything the
  plugin wrote outside `~/.local/share/lerd/plugins/<name>`, its data folder and
  the site it works on is a finding.
- **Network**: only loopback to Lerd, unless the README names more and the user
  opted in.
- **Lerd unbroken**: `md5sum -c /tmp/config-before`, the sites' status against
  `/tmp/sites-before`, `lerd status`, the dashboard loads, other plugins still
  load and `lerd plugin list` shows no failures.

## 4. Then a real machine

Only after the VM run is clean: install it, exercise its main surfaces once
against a real site, and remove it again if it is not wanted.

## 5. Report

Lead with the verdict: safe or not, breaks Lerd or not, follows the rules or
not. Then findings by term number, what the VM run showed for each surface,
and what could not be run. Draft the review comment as plain prose and wait
before posting it. When a plugin works around something Lerd's API lacks, say
which Lerd issue or pull request covers it.
