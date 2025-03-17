# Traefik WebFinger Plugin

![Traefik WebFinger](https://raw.githubusercontent.com/nx211/traefik-webfinger/main/.assets/icon.svg)

A [Traefik](https://traefik.io/) middleware plugin that implements the [WebFinger Protocol (RFC 7033)](https://datatracker.ietf.org/doc/html/rfc7033) for service discovery.

## What is WebFinger?

WebFinger is a protocol that enables the discovery of information about people or other entities on the Internet that are identified by a URI. WebFinger is used by federated social networks like Mastodon and other systems that need to discover services associated with a domain or email-like identifier.

## Features

- Simple integration with Traefik
- Configurable WebFinger resources via static configuration
- Option to pass through to backend services for dynamic WebFinger resources
- Follows the WebFinger specification (RFC 7033)
- Minimal impact on non-WebFinger requests

## Configuration

### Static Configuration

To use this plugin, you need to add it to your Traefik static configuration:

```yaml
# Static configuration
pilot:
  token: "xxxx"

experimental:
  plugins:
    webfinger:
      moduleName: "github.com/nx211/traefik-webfinger"
      version: "v0.1.0"
```

### Middleware Configuration

Then, you can configure the middleware in your dynamic configuration:

```yaml
# Dynamic configuration

http:
  middlewares:
    webfinger-handler:
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

### Configuration Options

- **domain**: (Required) The domain name this WebFinger service is responsible for.
- **resources**: (Optional) Map of WebFinger resources to serve statically.
  - Each resource is identified by a key (e.g., `acct:user@example.com`)
  - Each resource can have:
    - **subject**: (Required) The subject of the resource.
    - **aliases**: (Optional) List of URI aliases for the resource.
    - **links**: (Optional) List of links associated with the resource.
      - **rel**: (Required) The relation type of the link.
      - **type**: (Optional) The media type of the target resource.
      - **href**: (Optional) The URI of the target resource.
      - **titles**: (Optional) Human-readable labels for the link.
      - **properties**: (Optional) Additional properties of the link.
- **passthrough**: (Optional, default: false) Whether to pass through WebFinger requests to the backend service when the resource is not found.

## Integration with Traefik

Attach the middleware to a router in your dynamic configuration:

```yaml
http:
  routers:
    webfinger-router:
      rule: "Host(`example.com`) && Path(`/.well-known/webfinger`)"
      service: "your-backend"
      middlewares:
        - "webfinger-handler"
```

## Use Cases

### Federated Social Networks (Mastodon, Pleroma, etc.)

WebFinger is used by ActivityPub-based social networks to discover user accounts:

```yaml
resources:
  "acct:user@example.com":
    subject: "acct:user@example.com"
    links:
      - rel: "self"
        type: "application/activity+json"
        href: "https://example.com/users/user"
```

### OpenID Connect Provider Discovery

WebFinger can be used to discover OpenID Connect providers:

```yaml
resources:
  "acct:user@example.com":
    subject: "acct:user@example.com"
    links:
      - rel: "http://openid.net/specs/connect/1.0/issuer"
        href: "https://auth.example.com"
```

## License

This project is licensed under the [Apache License 2.0](LICENSE). See the [LICENSE](LICENSE) file for details.
