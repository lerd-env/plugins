# CLAUDE.md

This repository is Lerd's plugin catalog: one approved commit per plugin,
published at plugins.lerd.sh. The terms every plugin is held to are in
`SUBMITTING.md`; Lerd's rules for writing plugins are its guide,
`internal/plugins/guide.md` in lerd-env/lerd.

- A submission changes only its own `plugins/<name>.yaml`. Never touch another
  entry, the site or CI in the same pull request; the `scope` check rejects it.
- Pin the full 40-character commit, never a branch or tag. An update moves both
  the commit and a higher `version` in the plugin's `plugin.yaml`.
- To submit or update a plugin, use the `submit-plugin` skill. To check one, use
  `review-plugin`, and run nothing from a plugin on a real machine before its VM
  run is clean.
- Pull request text is plain prose following the template; nothing goes to
  GitHub without the person you work for saying so.

For maintainer work: the submission check, the index and the registry are Go
under `ci/` (`go test ./ci/...`), and `site/` is plugins.lerd.sh itself.
