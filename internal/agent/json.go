package agent

import "encoding/json"

var kindNames = map[EventKind]string{
	EvSession: "session", EvText: "text", EvThinking: "thinking", EvToolStart: "tool_start",
	EvToolResult: "tool_result", EvContext: "context", EvTodos: "todos", EvToolInput: "tool_input", EvArtifact: "artifact", EvDone: "done",
	EvError: "error", EvApproval: "approval",
}

// EventJSON renders an event as one JSON line for `nox -p --output json`.
func EventJSON(ev Event) string {
	o := map[string]any{"type": kindNames[ev.Kind]}
	put := func(k string, v any, ok bool) {
		if ok {
			o[k] = v
		}
	}
	put("text", ev.Text, ev.Text != "")
	put("tool", map[string]any{"id": ev.Tool.ID, "name": ev.Tool.Name, "input": ev.Tool.Input}, ev.Tool.ID != "")
	put("result", ev.Result, ev.Result != "")
	put("is_error", true, ev.IsError)
	put("session_id", ev.SessionID, ev.SessionID != "")
	put("model", ev.Model, ev.Model != "")
	put("context_window", ev.ContextWindow, ev.ContextWindow > 0)
	u := ev.Usage
	put("usage", map[string]any{"input": u.InputTokens, "output": u.OutputTokens, "cache_read": u.CacheRead, "cache_write": u.CacheWrite},
		u.InputTokens+u.OutputTokens+u.CacheRead > 0)
	put("todos", ev.Todos, len(ev.Todos) > 0)
	put("artifact", ev.Artifact, ev.Artifact != nil)
	if ev.Approval != nil {
		o["approval"] = map[string]any{"tool": ev.Approval.Tool.Name, "input": ev.Approval.Tool.Input, "reason": ev.Approval.Reason}
	}
	if ev.Err != nil {
		o["error"] = ev.Err.Error()
	}
	b, _ := json.Marshal(o)
	return string(b)
}
