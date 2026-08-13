# Specs

## Active

*In-flight specifications (DRAFT through BLOCKED) under `specs/active/`. Claim, block, unblock, park, and close via [`@rethunk/citadel-sdd`](https://github.com/Rethunk-AI/citadel-sdd).*

| Slug | State | DTG | Owner |
| ------ | ------- | ----- | ------- |
| *(none)* | | | |

## Parked

*Deliberately not pursued (**PARKED**); superseded or withdrawn specs under `specs/parked/`. Use `spec_park` from [`@rethunk/citadel-sdd`](https://github.com/Rethunk-AI/citadel-sdd).*

| Slug | DTG | Note |
| ------ | ----- | ------ |
| cli-account-export | 091233ZMAY26 | REJECTED — out of scope per NCA order 091232ZMAY26. GDPR data export is a web account-settings surface; CLI export path not in current phase mandate. |
| cli-billing | 091233ZMAY26 | REJECTED — out of scope per NCA order 091232ZMAY26. Billing management is a web/dashboard surface; CLI billing commands not in current phase mandate. |
| cli-webhook-test | 091233ZMAY26 | REJECTED — out of scope per NCA order 091232ZMAY26. Webhook test-ping is an operational surface; no current CLI mandate to wrap it. |
| cli-account-avatar | 072200ZMAY26 | Settings-panel concern, not a dev-loop workflow. No GitHub CLI analogue. Avatar management belongs in a browser/UI; a terminal user would never reach for this during a coding session. Superseded by the CLI-as-workflow-tool principle. |
| cli-account-privacy | 072200ZMAY26 | Settings-panel concern, not a dev-loop workflow. No GitHub CLI analogue. Privacy preference toggles belong in a browser/UI settings surface; they are not actions a developer would need mid-session. Superseded by the CLI-as-workflow-tool principle. |
| cli-mcp-stdio | 050505ZMAY26 | superseded by HTTPS MCP canonical policy ([`../README.md`](../README.md)). |
| cli-mcp-stream | 050505ZMAY26 | superseded by HTTPS MCP canonical policy ([`../README.md`](../README.md)). |
