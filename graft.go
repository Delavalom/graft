// Package graft is a Go-native framework for building AI agents and LLM-powered applications.
//
// Graft provides agent orchestration, multi-provider abstraction, tool execution,
// streaming, guardrails, lifecycle hooks, and OpenTelemetry observability.
//
// Basic usage:
//
//	agent := graft.NewAgent("assistant",
//	    graft.WithInstructions("You are a helpful assistant."),
//	    graft.WithTools(myTool),
//	)
//	runner := graft.NewDefaultRunner(model)
//	result, err := runner.Run(ctx, agent, messages)
package graft
