// Package graft is a Go framework for building AI agents and LLM-powered applications.
//
// Graft provides type-safe tool definitions, multi-provider LLM support (OpenAI, Anthropic,
// Google Gemini, AWS Bedrock), agent handoffs, guardrails, MCP integration, graph orchestration,
// and durable execution — all with zero vendor SDK dependencies.
//
// # Quick Start
//
//	agent := graft.NewAgent("assistant",
//	    graft.WithInstructions("You are a helpful assistant."),
//	    graft.WithTools(myTool),
//	)
//	runner := graft.NewDefaultRunner(model)
//	result, err := runner.Run(ctx, agent, messages)
//
// See https://github.com/delavalom/graft for full documentation and examples.
package graft
