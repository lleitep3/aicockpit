package personalagent

import "testing"

func TestMigrateConversationAddsSafeDefaults(t *testing.T) {
	for _, version := range []int{1, 2, CurrentSchemaVersion} {
		t.Run(versionName(version), func(t *testing.T) {
			conversation := Conversation{
				SchemaVersion: version,
				Turns:         []Turn{{ID: "turn_old"}},
			}
			if err := migrateConversation(&conversation); err != nil {
				t.Fatalf("migrateConversation() error = %v", err)
			}
			if conversation.SchemaVersion != CurrentSchemaVersion {
				t.Fatalf("schema version = %d, want %d", conversation.SchemaVersion, CurrentSchemaVersion)
			}
			if conversation.Selection.ApprovalPolicy != "on-request" {
				t.Fatalf("selection approval policy = %q, want on-request", conversation.Selection.ApprovalPolicy)
			}
			if conversation.Selection.RoutingMode != RoutingModeManual {
				t.Fatalf("selection routing mode = %q, want manual", conversation.Selection.RoutingMode)
			}
			if conversation.Turns[0].ApprovalPolicy != "on-request" {
				t.Fatalf("turn approval policy = %q, want on-request", conversation.Turns[0].ApprovalPolicy)
			}
			if conversation.Turns[0].RoutingMode != RoutingModeManual {
				t.Fatalf("turn routing mode = %q, want manual", conversation.Turns[0].RoutingMode)
			}
		})
	}
}

func TestMigrateConversationRejectsUnknownVersion(t *testing.T) {
	conversation := Conversation{SchemaVersion: CurrentSchemaVersion + 1}
	if err := migrateConversation(&conversation); err == nil {
		t.Fatal("migrateConversation() error = nil, want unsupported schema error")
	}
}

func versionName(version int) string {
	if version == CurrentSchemaVersion {
		return "current"
	}
	if version == 1 {
		return "v1"
	}
	return "v2"
}
