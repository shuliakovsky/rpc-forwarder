##  Environment Variables

The RPC Forwarder uses several environment variables to configure its behavior. Here's a complete list:

| Variable Name             | Description                                                    | Default Value       |
|---------------------------|----------------------------------------------------------------|---------------------|
| `SERVER_HOST`             | Host address to bind the HTTP server                           | `0.0.0.0`           |
| `SERVER_PORT`             | Port to bind the HTTP server                                   | `8080`              |
| `POD_IP`                  | Internal IP of the node (used for gossip/bootstrap)            | `127.0.0.1`         |
| `POD_NAME`                | Node name (used for gossip/bootstrap)                          | `dev-node`          |
| `SHARED_SECRET`           | Shared secret for bootstrap signature validation               | `devsecret`         |
| `BOOTSTRAP_URL`           | Optional URL of a bootstrap node                               | *(empty)*           |
| `TOR_SOCKS5`              | SOCKS5 proxy address for Tor-enabled nodes                     | `127.0.0.1:9050`    |
| `ADMIN_API_KEY`           | API key for accessing `/admin/*` endpoints                     | `changeme`          |
| `SWAGGER_HOST`            | Hostname for Swagger UI                                        | *(optional)*        |
| `TATUM_API_KEY`           | API key for Tatum RPC providers                                | *(required)*        |
| `TATUM_API_KEY_TESTNET`   | Optional testnet key for Tatum (now properly redacted in logs) | *(optional)*        |
| `ALCHEMY_API_KEY`         | API key for Alchemy RPC providers                              | *(required)*        |
| `ALCHEMY_API_KEY_TESTNET` | Optional testnet key for Alchemy RPC providers                 | *(required)*        |

> ️ If `ADMIN_API_KEY` is left as `changeme`, admin endpoints are unprotected.

---

##  Secrets & Redaction

### Secret Injection from Environment

When loading network configurations from `configs/networks/*.yaml`, any value written as `${VAR_NAME}` will be automatically replaced at startup using `os.Getenv("VAR_NAME")`.

- If the environment variable is missing, a warning will be logged.
- The placeholder will be replaced with an empty string.

Example:

```yaml
headers:
  x-api-key: ${TATUM_API_KEY}
