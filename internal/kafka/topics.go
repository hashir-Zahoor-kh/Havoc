// Package kafka provides thin wrappers around the Kafka client used by the
// control plane (producer), the agent (consumer + producer), and the
// recorder (consumer).
package kafka

// TopicCommands carries experiment commands from the control plane to agents.
const TopicCommands = "havoc.commands"

// TopicResults carries agent-reported experiment outcomes to the recorder.
const TopicResults = "havoc.results"
