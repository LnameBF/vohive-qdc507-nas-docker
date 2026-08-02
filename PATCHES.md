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
