# Traefik WebFinger Plugin

![Traefik WebFinger](https://raw.githubusercontent.com/nx211/traefik-webfinger/main/.assets/logo.svg)

A [Traefik](https://traefik.io/) middleware plugin that implements the [WebFinger Protocol (RFC 7033)](https://datatracker.ietf.org/doc/html/rfc7033) for service discovery.

## Features

- Full WebFinger protocol support according to RFC 7033
- Static resource configuration
- Domain-based resource filtering
- Optional passthrough to backend services
- Support for multiple resource types (acct:, https://, mailto:)
- Configurable aliases and links
- JRD+JSON response format

## Installation

To configure the WebFinger plugin in your Traefik instance:

1. Enable the plugin in your static configuration:

```yaml
# Static configuration
experimental:
  plugins:
    webfinger:
      moduleName: github.com/NX211/traefik-webfinger
      version: v0.1.0
```

2. Configure the middleware in your dynamic configuration:

```yaml
# Dynamic configuration
http:
  middlewares:
    my-webfinger:
      plugin:
        webfinger:
          domain: "example.com"
          resources:
            "acct:user@example.com":
              subject: "acct:user@example.com"
              aliases:
                - "https://example.com/user"
              links:
                - rel: "http://webfinger.net/rel/profile-page"
                  type: "text/html"
                  href: "https://example.com/user"
                - rel: "self"
                  type: "application/activity+json"
                  href: "https://example.com/users/user"
          passthrough: false
```

3. Use the middleware in your router:

```yaml
http:
  routers:
    my-router:
      rule: "Host(`example.com`)"
      service: my-service
      middlewares:
        - my-webfinger
```

## Configuration

### Plugin Configuration Options

| Option | Type | Required | Default | Description |
|--------|------|----------|---------|-------------|
| domain | string | Yes | "" | The domain this WebFinger service handles |
| resources | map | No | {} | Map of WebFinger resources and their responses |
| passthrough | bool | No | false | Whether to pass through to backend when resource not found |

### Resource Configuration

Each resource in the `resources` map can have the following properties:

| Property | Type | Required | Description |
|----------|------|----------|-------------|
| subject | string | Yes | The resource identifier |
| aliases | []string | No | Alternative identifiers for the resource |
| links | []Link | No | Related links for the resource |

### Link Configuration

Each link in the `links` array can have:

| Property | Type | Required | Description |
|----------|------|----------|-------------|
| rel | string | Yes | The link relation type |
| type | string | No | The content type of the linked resource |
| href | string | No | The URL of the linked resource |
| titles | map[string]string | No | Titles in different languages |
| properties | map[string]string | No | Additional properties |

## Example Usage

### Basic Configuration

```yaml
http:
  middlewares:
    webfinger:
      plugin:
        webfinger:
          domain: "example.com"
          resources:
            "acct:alice@example.com":
              subject: "acct:alice@example.com"
              aliases:
                - "https://example.com/alice"
              links:
                - rel: "http://webfinger.net/rel/profile-page"
                  type: "text/html"
                  href: "https://example.com/alice"
```

### Making Requests

To query a WebFinger resource:

```bash
curl "https://example.com/.well-known/webfinger?resource=acct:alice@example.com"
```

Example response:

```json
{
  "subject": "acct:alice@example.com",
  "aliases": [
    "https://example.com/alice"
  ],
  "links": [
    {
      "rel": "http://webfinger.net/rel/profile-page",
      "type": "text/html",
      "href": "https://example.com/alice"
    }
  ]
}
```

## Development

To build and test the plugin:

```bash
# Run tests
go test ./...

# Build the plugin
go build ./...
```

### Local Development

For local development, you can use Traefik's plugin development mode:

```yaml
# Static configuration
experimental:
  localPlugins:
    webfinger:
      moduleName: github.com/NX211/traefik-webfinger
```

## License

This project is licensed under the [Apache License 2.0](LICENSE). See the [LICENSE](LICENSE) file for details.
