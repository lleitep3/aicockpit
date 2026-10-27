package personalagent

import "fmt"

func migrateConversation(conversation *Conversation) error {
	switch conversation.SchemaVersion {
	case 1, 2:
		conversation.Selection.ApprovalPolicy = defaultApprovalPolicy(conversation.Selection.ApprovalPolicy)
		conversation.Selection.RoutingMode = defaultRoutingMode(conversation.Selection.RoutingMode)
		conversation.SchemaVersion = CurrentSchemaVersion
	case CurrentSchemaVersion:
		conversation.Selection.ApprovalPolicy = defaultApprovalPolicy(conversation.Selection.ApprovalPolicy)
		conversation.Selection.RoutingMode = defaultRoutingMode(conversation.Selection.RoutingMode)
	default:
		return fmt.Errorf("unsupported conversation schema version %d", conversation.SchemaVersion)
	}
	for index := range conversation.Turns {
		conversation.Turns[index].ApprovalPolicy = defaultApprovalPolicy(conversation.Turns[index].ApprovalPolicy)
		conversation.Turns[index].RoutingMode = defaultRoutingMode(conversation.Turns[index].RoutingMode)
	}
	if conversation.ParentConversationID != "" && conversation.RootConversationID == "" {
		conversation.RootConversationID = conversation.ParentConversationID
	} else if conversation.RootConversationID != "" && conversation.ParentConversationID == "" {
		conversation.ParentConversationID = conversation.RootConversationID
	}
	return nil
}

func defaultRoutingMode(mode string) string {
	if mode == "" {
		return RoutingModeManual
	}
	return mode
}

func defaultApprovalPolicy(policy string) string {
	if policy == "" {
		return "on-request"
	}
	return policy
}
