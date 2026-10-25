# K8s Admission Webhook Last Updated

[![Tag](https://img.shields.io/github/tag/tschaefer/last-updated.svg)](https://github.com/tschaefer/last-updated/releases)
[![Contributors](https://img.shields.io/github/contributors/tschaefer/last-updated)](https://github.com/tschaefer/last-updated/graphs/contributors)
[![License](https://img.shields.io/github/license/tschaefer/last-updated)](./LICENSE)

Mutating Admission Webhook to annotate Kubernetes resources with a last updated
timestamp.

# Getting Started

## Kubernetes deployment

Deploy the latest release in namespace `webhook-system`.

> [!NOTE]
> This requires [cert-manager](https://cert-manager.io) to be installed for
> issuing the webhook TLS assets.

```bash
kubectl apply -f \
    https://github.com/tschaefer/last-updated/releases/download/0.6.1/install.yaml
```

For details and customized setup concider the manifests.

## CLI Flags

```bash
Usage of last-updated:
  -config.path string
        path to webhook configuration
  -log.level string
        log level (default: info)
  -profiler.address string
        profiler server address (default "http://pyroscope:4040")
  -profiler.enable string (default "false")
        enable Pyroscope push profiler
  -tls.cert string
        path to TLS certificate
  -tls.key string
        path to TLS key
  -version
        show version
```

The flags `-config.path` (see [Webhook Configuration](#webhook-configuration)),
`-tls.cert` and `-tls.key` are mandatory. All settings may also be provided
by the related environment variables.

- `CONFIG_PATH`
- `LOG_LEVEL`
- `PROFILER_ADDRESS`
- `PROFILER_ENABLE`
- `TLS_CERT`
- `TLS_KEY`

Flags keep precedence over environment variables.

To enable continous profiling the `-profiler.enable` flag value requires to be
one of `YES`, `yes`, `TRUE`, `true` or `1`.

## Webhook configuration

The yaml formatted webhook configuration describes following fields.

| Key | Description | Required |
|-----|-------------|----------|
| `annotationKey` | key name of the annotation | true |
| `timeStampFormat` | timestamp format of the annotation `RFC3339` / `RFC339Nano` | true |
| `updateOnCreate` | annotation on resource creation | false |
| `excludeSubresources` | exclude annotation if subresource is modified | false |
| `resources` | annotation on list of resources described by resource `group` and `resource` name | one resource |
| `excludedNamespaces` | exclude annotation of resources in list of namespaces | false |

The webhook service requires permission to `create` and `update` annotations in
the configured resources. The required `MutatingWebhookConfiguration` should
for security reason exclude namespaces via selector already, and set rules only
for configured resources.

# Contributing

Fork the repository and submit a pull request. For major changes, open an issue
first to discuss your proposal.

# License

This project is licensed under the [MIT License](LICENSE).
