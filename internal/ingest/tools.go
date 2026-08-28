package ingest

import (
	"encoding/json"
	"strings"
)

func parseTool(m map[string]any) toolUse {
	t := toolUse{Name: asString(m["name"])}
	if t.Name == "" {
		t.Name = asString(m["toolName"])
	}
	switch in := m["input"].(type) {
	case map[string]any:
		t.Input = in
	case string:
		var obj map[string]any
		if json.Unmarshal([]byte(in), &obj) == nil {
			t.Input = obj
		}
	}
	t.Path = toolPath(t)
	t.Delete = isDeleteTool(t.Name)
	return t
}

func toolPath(t toolUse) string {
	if t.Input == nil {
		return ""
	}
	for _, k := range []string{"file_path", "path", "target_path", "target_file", "targetFile", "notebook_path", "target"} {
		if p := asString(t.Input[k]); p != "" {
			return strings.TrimPrefix(p, "file://")
		}
	}
	if p := asString(t.Input["uri"]); strings.HasPrefix(p, "file://") {
		return strings.TrimPrefix(p, "file://")
	}
	if isMutateTool(t.Name) {
		return ""
	}
	if strings.EqualFold(t.Name, "Bash") || strings.EqualFold(t.Name, "Shell") {
		return bashPath(asString(t.Input["command"]))
	}
	return ""
}

func isMutateTool(name string) bool {
	switch strings.ToLower(name) {
	case "edit", "write", "edit_file", "notebookedit", "editnotebook", "strreplace",
		"search_replace", "searchreplace", "applypatch", "apply_patch",
		"delete", "delete_file", "deletefile", "remove":
		return true
	}
	return false
}

func isDeleteTool(name string) bool {
	switch strings.ToLower(name) {
	case "delete", "delete_file", "deletefile", "remove":
		return true
	}
	return false
}

func isEditWrite(name string) bool {
	switch strings.ToLower(name) {
	case "edit", "write", "edit_file", "notebookedit", "editnotebook", "strreplace",
		"search_replace", "searchreplace":
		return true
	}
	return false
}
