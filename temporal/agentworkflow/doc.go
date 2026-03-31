// Package agentworkflow provides a default Temporal workflow implementation
// for running graft agents with per-step durability.
//
// Each LLM call and tool execution runs as a separate Temporal activity,
// giving fine-grained retry and recovery. On worker crash, only the failed
// step reruns — completed LLM calls and tool results are preserved.
//
// # Worker Setup
//
//	toolRegistry := agentworkflow.NewToolRegistry()
//	toolRegistry.Register(myTool1, myTool2)
//	models := agentworkflow.NewSingleModelProvider(myModel)
//
//	w := worker.New(c, "ai-agents", worker.Options{})
//	agentworkflow.RegisterAgentWorkflow(w)
//	agentworkflow.RegisterAgentActivities(w, models, toolRegistry)
//	w.Run(worker.InterruptCh())
//
// # Client Setup
//
// Use the existing [github.com/delavalom/graft/temporal.TemporalRunner] to
// start workflows from the client side. The runner submits WorkflowInput
// which DefaultAgentWorkflow picks up on the worker.
package agentworkflow
