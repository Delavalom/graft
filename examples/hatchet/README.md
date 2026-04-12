# Hatchet example

Two sub-packages, one for each side of a Hatchet deployment:

| Directory | Purpose | Runnable? |
|-----------|---------|-----------|
| `client/` | Dispatch an agent run through `hatchet.HatchetRunner`. | Yes — uses an in-process demo client so you can see the flow without infrastructure. |
| `worker/` | Register the agent-loop DAG handlers on a Hatchet worker. | Compiles and prints the DAG, but does nothing real until you wire the handlers to `hatchet-go`. |

## Run the client

```bash
go run ./examples/hatchet/client
```

Output is logged by the demo client (one `RunWorkflow` call) and the canned assistant response is printed.

## Promote to a real Hatchet deployment

1. Add the Hatchet SDK to your module:
   ```bash
   go get github.com/hatchet-dev/hatchet-go
   ```
2. In `client/main.go`, replace `demoClient{}` with a real `*hatchet.Client`:
   ```go
   c, err := hatchetclient.New()
   if err != nil { log.Fatal(err) }
   runner := hatchet.NewHatchetRunner(c, ...)
   ```
3. In `worker/main.go`, register the three step handlers (`llm-generate`, `tool-execute`, `loop-check`) on a real Hatchet worker and start it.

See [docs: runtimes/hatchet](../../../website/content/docs/runtimes/hatchet.mdx) for the concepts.
