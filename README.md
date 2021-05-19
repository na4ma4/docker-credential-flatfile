# docker-credential-flatfile

Flatfile Docker Credential Helper

__WARNING__: This is severely and really insecure, it stores the secrets in plain text, not even obscured.

This is for CI systems where the `docker login` command can't use the system keystores (`macOS`, etc).

## Usage

Download the latest release and put it in `/usr/local/bin/`, change the `credsStore` in `~/.docker/config.json` to `flatfile`.

Example:

```json
{
  "auths": {},
  "credsStore": "flatfile",
  "experimental": "disabled",
  "stackOrchestrator": "swarm"
}
```
