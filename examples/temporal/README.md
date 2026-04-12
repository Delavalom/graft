# Temporal example

Two sub-packages, one for each side of a Temporal deployment:

| Directory | Purpose | Runnable? |
|-----------|---------|-----------|
| `client/` | Start an agent workflow via `temporal.TemporalRunner`. | Yes — uses an in-process demo client so you can see the flow without infrastructure. |
| `worker/` | Register `AgentWorkflow` and its activities on a Temporal worker. | Compiles and prints a status line, but is pseudocode until you add `go.temporal.io/sdk`. |

## Run the client

```bash
go run ./examples/temporal/client
```

Output is logged by the demo client (one `ExecuteWorkflow` call) and the canned assistant response is printed.

## Promote to a real Temporal deployment

1. Add the Temporal SDK to your module:
   ```bash
   go get go.temporal.io/sdk
   ```
2. In `client/main.go`, replace `demoClient{}` with a real `*client.Client`:
   ```go
   c, err := client.Dial(client.Options{})
   if err != nil { log.Fatal(err) }
   defer c.Close()
   runner := temporal.NewTemporalRunner(c, ...)
   ```
3. In `worker/main.go`, register `AgentWorkflow` + activities using `worker.New` and run the worker.

See [docs: runtimes/temporal](../../../website/content/docs/runtimes/temporal.mdx) for the concepts.
