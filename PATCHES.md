# Local build patches

The bundled VoHive source retains its original PolyForm Noncommercial License
and required notice.  The following narrow changes are applied solely so this
repository can build reproducibly in a Docker environment:

- Configuration persistence imports use the already-required
  `gopkg.in/yaml.v3` module instead of the unreachable `go.yaml.in/yaml/v3`
  mirror.
- Test-only and unused indirect requirements are omitted from the bundled
  `go.mod`; runtime source code is unchanged by those removals.
- The root module explicitly maps the bundled `swu-go` source tree so a
  Docker build never attempts to fetch the unavailable upstream placeholder
  revision.

These patches do not add firmware, credentials, device identities, or private
network configuration.

## Behavioral patches

The following changes alter upstream runtime behavior, not just build output.
They make the end-user disclaimer/EULA prompt appear only on first use across
all devices, instead of upstream's "every login" behavior (upstream stores the
"agreed" flag in `sessionStorage` and clears it on logout):

- Backend (`internal/api/system.go`, `internal/api/server.go`): add two
  authenticated endpoints — `GET /api/system/disclaimer` reports whether the
  disclaimer has been accepted, and `POST /api/system/disclaimer/accept`
  records acceptance by creating a marker file at `data/.disclaimer_accepted`.
  The marker lives in the persisted `data/` volume (the same directory the
  reject/self-destruct flow wipes), so acceptance survives restarts and is
  shared by every client/device. After a reject+reinstall the marker is gone,
  so first-use correctly re-prompts.
- Frontend (`web/src/App.vue`): the disclaimer state is now queried from the
  server (via the shared authenticated `api` client) instead of client-side
  `sessionStorage`. On login the UI asks the server whether the disclaimer was
  already accepted; if so, no prompt. Acceptance is persisted server-side.
  The reject-and-uninstall path is unchanged.
