---
name: submit-plugin
description: Submit a Lerd plugin to the catalog, or an update to one already listed, by checking it against the terms first and then writing its entry and pull request. Use when asked to submit, list, publish or update a plugin on plugins.lerd.sh.
---

# Submit a plugin

A submission is one file, `plugins/<name>.yaml`, and a pull request a
maintainer reviews against `SUBMITTING.md`. Read `SUBMITTING.md` and
`.github/pull_request_template.md` first; this skill applies them, it does not
replace them.

## 1. Review it yourself first

Run the `review-plugin` skill on the plugin at the commit you mean to submit.
Fix what it finds and commit again before going on: a maintainer runs the same
review, and every finding left in costs a round trip.

## 2. Pin the commit

- The commit must be pushed to the plugin's public repository, and it is the
  exact code Lerd will install. Use the full 40-character hash
  (`git rev-parse HEAD` after pushing), never a branch or tag.
- For an update, both must move: the new commit, and a higher `version` in
  `plugin.yaml` than the commit currently listed. Bump the version in the
  plugin's repository, push, and pin that commit.
- `plugin.yaml` declares `lerd` as a lower bound no older than the first
  release with what it uses (plugins exist from 1.38), and `platforms` when it
  runs on only one.

## 3. Write the entry

`plugins/<name>.yaml`, named like the plugin:

```yaml
repo: https://github.com/<owner>/<repo>
commit: <full 40-character commit>
author: <who it is listed by>
description: <one line, what it does for the user>
category: <one the catalog already uses, see the other entries>
tags: [<a few lowercase words>]
images: [docs/<file>.png]   # paths in the repository at that commit, the first is the preview
```

- A plugin with a page, site tab, Overview card or page script lists at least
  one image: PNG, JPEG or WebP, under 1 MB, at most 2560 px wide, at most six.
- The pull request changes only this file.

## 4. Write the pull request

Follow the template: a few short paragraphs on what it does and adds, what it
reads, changes and sends (and to whom), what it needs, and for an update the
previously approved commit and what changed since. Then the repository line and
each image linked at the commit:

```
![What it shows](https://raw.githubusercontent.com/<owner>/<repo>/<commit>/<path>)
```

Tick the checklist honestly; an unticked box with a reason is better than a
ticked one that is not true.

## 5. Open it

Show the entry and the description to the person you are working for and open
the pull request only once they say so. Then leave it: the `check` job runs on
its own and a maintainer reviews it.
