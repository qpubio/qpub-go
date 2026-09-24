# qpub-go documentation

| Document                                               | Purpose                                                                       |
| ------------------------------------------------------ | ----------------------------------------------------------------------------- |
| [../CONTRIBUTING.md](../CONTRIBUTING.md)               | Develop, run examples, test, lint, debug, PR workflow                         |
| [architecture.md](./architecture.md)                   | Layer model, import rules, package map (Go-idiomatic, qpub-js-aligned spirit) |
| [implementation-status.md](./implementation-status.md) | Parity plan phases 0–8: done, partial, and remaining work                     |
| [cross-sdk-api.md](./cross-sdk-api.md)                 | JavaScript vs Go public API comparison                                        |

When adding or moving code, read **architecture.md** first. When prioritizing parity work, use **implementation-status.md** as the single checklist (do not duplicate phase tracking elsewhere).

Examples: [basic REST](../examples/basic/), [socket](../examples/socket/), [token auth](../examples/token-auth/), [queue worker](../examples/queue-worker/).
