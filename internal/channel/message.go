// Package channel manages the lifecycle of channel plugins and routes messages to the agent.
package channel

import "github.com/gopaw/gopaw/pkg/types"

// re-export types for convenience so callers can use channel.Message etc.
type Message = types.Message
type FileAttachment = types.FileAttachment
type MessageType = types.MessageType
