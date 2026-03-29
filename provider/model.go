// Package provider implements LLM provider abstractions for the graft framework.
// It re-exports the LanguageModel interface and associated types from the root
// graft package for convenience, and provides routing and middleware utilities.
package provider

import (
	"github.com/delavalom/graft"
)

// LanguageModel is the interface all LLM providers must implement.
// It is an alias for graft.LanguageModel.
type LanguageModel = graft.LanguageModel

// GenerateParams holds the parameters for a generation request.
// It is an alias for graft.GenerateParams.
type GenerateParams = graft.GenerateParams

// GenerateResult holds the result of a generation request.
// It is an alias for graft.GenerateResult.
type GenerateResult = graft.GenerateResult

// StreamChunk holds a single chunk of a streaming response.
// It is an alias for graft.StreamChunk.
type StreamChunk = graft.StreamChunk
