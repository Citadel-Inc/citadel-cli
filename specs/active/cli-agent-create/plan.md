# Plan — cli-agent-create

## API survey (P0 task A1)

Before implementation, confirm the `POST /api/agents` request body and response shape. Based on `cli-oauth-login` spec §21:

```
POST /api/agents
{ "name": "citadel-cli@hostname" }
→ { "id": "...", "name": "...", "token": "..." }
```

Questions to resolve:
1. Does the endpoint accept `org_slug` to create an agent under an org namespace?
2. Is the initial token returned on create, or does a separate `rotate-token` call issue the first token?
3. What validation constraints apply to `name` (length, charset, uniqueness scope)?

## CLI shape

```go
var agentCreateCmd = &cobra.Command{
    Use:   "create <name>",
    Short: "Register a new agent",
    Args:  cobra.ExactArgs(1),
    RunE:  runAgentCreate,
}

func init() {
    agentCreateCmd.Flags().String("org", "", "org namespace to create the agent under")
    agentCreateCmd.Flags().String("description", "", "optional agent description")
    AgentCmd.AddCommand(agentCreateCmd)
}

func runAgentCreate(cmd *cobra.Command, args []string) error {
    name := args[0]
    org, _ := cmd.Flags().GetString("org")

    c, err := newAPIClient(cmd)
    if err != nil { return err }

    req := createAgentRequest{Name: name, OrgSlug: org}
    var resp agentCreateResponse
    if err := c.Post(cmd.Context(), "/agents", req, &resp); err != nil {
        if apiclient.IsStatus(err, http.StatusConflict) {
            return fmt.Errorf("agent name %q is already taken", name)
        }
        if apiclient.IsStatus(err, http.StatusForbidden) {
            return fmt.Errorf("insufficient permission to create an agent in this namespace")
        }
        return err
    }

    if outputFlag(cmd) == "json" {
        return emitJSON(resp)
    }
    fmt.Fprintf(cmd.OutOrStdout(), "Agent created\n  ID:    %s\n  Name:  %s\n  Token: %s\n\n", resp.ID, resp.Name, resp.Token)
    fmt.Fprintln(cmd.ErrOrStderr(), "⚠  Save this token — it will not be shown again.")
    return nil
}
```

## Estimated delta

| Component | LOC (rough) |
|-----------|-------------|
| `createCmd` + flag wiring | 30 |
| `runAgentCreate` + error handling | 60 |
| `agentCreateRequest` / `agentCreateResponse` types | 20 |
| Tests (happy + 409 + 403) | 80 |
| Docs update | 20 |
| **Total** | **~210** |

## Risks

- **Org-scoped path**: if the daemon requires `POST /api/orgs/{slug}/agents` instead of a body field, the URL construction in `runAgentCreate` needs a branch. Survey resolves this.
- **Token on create vs. rotate**: if `POST /api/agents` does not return an initial token, the create verb must follow up with `rotate-token` immediately. Confirm in survey.
