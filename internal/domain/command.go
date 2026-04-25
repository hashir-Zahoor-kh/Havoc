package domain

// CommandType discriminates the messages on the havoc.commands topic.
type CommandType string

const (
	// CommandSchedule carries a fully-validated experiment for execution.
	CommandSchedule CommandType = "schedule"
	// CommandAbort tells agents to cancel a single in-flight experiment.
	CommandAbort CommandType = "abort"
	// CommandKillSwitch tells agents to abort every in-flight experiment.
	// It is a redundant signal — agents also check Redis directly — but
	// the broadcast lets them react without polling.
	CommandKillSwitch CommandType = "kill_switch"
)

// Command is the envelope every message on havoc.commands carries. Agents
// dispatch on Type. Fields not relevant to the type are zero.
type Command struct {
	Type         CommandType `json:"type"`
	Experiment   *Experiment `json:"experiment,omitempty"`
	ExperimentID ID          `json:"experiment_id,omitempty"`
}
