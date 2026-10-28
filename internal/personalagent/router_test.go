package personalagent

import (
	"strings"
	"testing"
)

func TestClassifyTaskUsesFirstMatchingClassAndReason(t *testing.T) {
	testCases := []struct {
		name      string
		content   string
		wantClass string
		wantHint  string
	}{
		{name: "quick", content: "Resuma este parágrafo", wantClass: TaskClassQuick},
		{name: "coding", content: "Implemente o endpoint de conversa", wantClass: TaskClassCoding, wantHint: "implement"},
		{name: "research", content: "Pesquise fontes atuais sobre o tema", wantClass: TaskClassResearch, wantHint: "pesquise"},
		{name: "reasoning", content: "Compare as opções de arquitetura", wantClass: TaskClassReasoning, wantHint: "arquitetura"},
		{name: "coding priority", content: "Pesquise e implemente a correção", wantClass: TaskClassCoding},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			classification := ClassifyTask(testCase.content, nil)
			if classification.TaskClass != testCase.wantClass {
				t.Fatalf("ClassifyTask() class = %q, want %q", classification.TaskClass, testCase.wantClass)
			}
			if testCase.wantHint != "" && !strings.Contains(classification.Reason, testCase.wantHint) {
				t.Fatalf("ClassifyTask() reason = %q, want it to contain %q", classification.Reason, testCase.wantHint)
			}
		})
	}
}

func TestClassifyTaskTreatsCodeAttachmentAsCoding(t *testing.T) {
	classification := ClassifyTask("Explique este arquivo", []Attachment{{Name: "main.go"}})
	if classification.TaskClass != TaskClassCoding {
		t.Fatalf("ClassifyTask() class = %q, want %q", classification.TaskClass, TaskClassCoding)
	}
	if !strings.Contains(classification.Reason, "arquivo de código") {
		t.Fatalf("ClassifyTask() reason = %q, want code attachment explanation", classification.Reason)
	}
}

func TestPrepareProviderContext(t *testing.T) {
	previous := []Message{{ID: "msg_previous", Role: "user", Content: "Pedido anterior"}}
	testCases := []struct {
		name          string
		conversation  Conversation
		providerID    string
		wantMode      string
		wantNativeID  string
		wantPromptHas string
		wantSourceIDs int
	}{
		{
			name:          "first turn",
			conversation:  Conversation{NativeSessions: map[string]string{}},
			providerID:    "codex",
			wantMode:      "none",
			wantPromptHas: "pedido atual",
		},
		{
			name: "new provider receives canonical history",
			conversation: Conversation{
				Messages:       previous,
				Turns:          []Turn{{ProviderID: "codex"}},
				NativeSessions: map[string]string{},
			},
			providerID:    "claude",
			wantMode:      "canonical_transcript",
			wantPromptHas: "Pedido anterior",
			wantSourceIDs: 1,
		},
		{
			name: "same native session continues",
			conversation: Conversation{
				Messages:       previous,
				Turns:          []Turn{{ProviderID: "codex"}},
				NativeSessions: map[string]string{"codex": "native-1"},
			},
			providerID:    "codex",
			wantMode:      "provider_native_session",
			wantNativeID:  "native-1",
			wantSourceIDs: 1,
		},
		{
			name: "returning to native session receives intervening history",
			conversation: Conversation{
				Messages:       previous,
				Turns:          []Turn{{ProviderID: "claude"}},
				NativeSessions: map[string]string{"codex": "native-1"},
			},
			providerID:    "codex",
			wantMode:      "provider_native_session+canonical_transcript",
			wantNativeID:  "native-1",
			wantPromptHas: "Pedido anterior",
			wantSourceIDs: 1,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			nativeID, mode, sourceIDs, prompt, err := prepareProviderContext(testCase.conversation, testCase.providerID, "pedido atual")
			if err != nil {
				t.Fatalf("prepareProviderContext() error = %v", err)
			}
			if mode != testCase.wantMode {
				t.Fatalf("prepareProviderContext() mode = %q, want %q", mode, testCase.wantMode)
			}
			if nativeID != testCase.wantNativeID {
				t.Fatalf("prepareProviderContext() native session = %q, want %q", nativeID, testCase.wantNativeID)
			}
			if len(sourceIDs) != testCase.wantSourceIDs {
				t.Fatalf("prepareProviderContext() source IDs = %d, want %d", len(sourceIDs), testCase.wantSourceIDs)
			}
			if testCase.wantPromptHas != "" && !strings.Contains(strings.ToLower(prompt), strings.ToLower(testCase.wantPromptHas)) {
				t.Fatalf("prepareProviderContext() prompt does not contain %q", testCase.wantPromptHas)
			}
		})
	}
}

func TestPrepareProviderContextRejectsOversizedTranscript(t *testing.T) {
	conversation := Conversation{
		Messages: []Message{{ID: "msg_large", Role: "user", Content: strings.Repeat("x", maximumCanonicalContextBytes)}},
	}
	_, _, _, _, err := prepareProviderContext(conversation, "codex", "current")
	if err == nil || !strings.Contains(err.Error(), "avoid dropping context") {
		t.Fatalf("prepareProviderContext() error = %v, want oversized transcript error", err)
	}
}
