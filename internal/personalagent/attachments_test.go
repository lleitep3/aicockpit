package personalagent

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestValidateAttachmentsSupportsSafeImageTypes(t *testing.T) {
	tests := []struct {
		name        string
		attachment  Attachment
		wantErrPart string
	}{
		{
			name:       "PNG screenshot",
			attachment: Attachment{Name: "screenshot.png", MediaType: "image/png", Content: base64.StdEncoding.EncodeToString([]byte("\x89PNG\r\n\x1a\n"))},
		},
		{
			name:        "unsupported SVG",
			attachment:  Attachment{Name: "image.svg", MediaType: "image/svg+xml", Content: "PHN2Zz4="},
			wantErrPart: "unsupported media type",
		},
		{
			name:        "mismatched signature",
			attachment:  Attachment{Name: "image.png", MediaType: "image/png", Content: base64.StdEncoding.EncodeToString([]byte("not a PNG"))},
			wantErrPart: "does not match its media type",
		},
		{
			name:        "invalid base64",
			attachment:  Attachment{Name: "image.png", MediaType: "image/png", Content: "%%%"},
			wantErrPart: "not valid base64",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateAttachments([]Attachment{test.attachment})
			if test.wantErrPart == "" {
				if err != nil {
					t.Fatalf("validateAttachments() error = %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), test.wantErrPart) {
				t.Fatalf("validateAttachments() error = %v, want it to contain %q", err, test.wantErrPart)
			}
		})
	}
}

func TestComposePromptOmitsImageDataAndPreservesTextFiles(t *testing.T) {
	imageData := base64.StdEncoding.EncodeToString([]byte("\x89PNG\r\n\x1a\n"))
	prompt, err := composePrompt("Analise os anexos", []Attachment{
		{Name: "notes.txt", MediaType: "text/plain", Content: "contexto"},
		{Name: "screenshot.png", MediaType: "image/png", Content: imageData},
	})
	if err != nil {
		t.Fatalf("composePrompt() error = %v", err)
	}
	if !strings.Contains(prompt, "contexto") {
		t.Fatal("composePrompt() omitted text attachment content")
	}
	if strings.Contains(prompt, imageData) {
		t.Fatal("composePrompt() leaked base64 image data into the text prompt")
	}
}

func TestCodexTurnInputIncludesInlineImage(t *testing.T) {
	input := codexTurnInput("O que aparece?", []Attachment{
		{Name: "screenshot.png", MediaType: "image/png", Content: "iVBORw0KGgo="},
	})
	if len(input) != 2 || input[0]["type"] != "text" || input[1]["type"] != "image" {
		t.Fatalf("codexTurnInput() = %#v, want text followed by image", input)
	}
	if input[1]["url"] != "data:image/png;base64,iVBORw0KGgo=" || input[1]["detail"] != "high" {
		t.Fatalf("image input = %#v, want inline high-detail data URL", input[1])
	}
}

func TestResolveImageSelectionRoutesToCodex(t *testing.T) {
	service, _ := newRoutingTestService(t)
	selection, classification, err := service.resolveImageSelection(
		Selection{ProviderID: "antigravity", ModelID: "gemini", ReasoningEffort: "high"},
		"Analise a captura",
		[]Attachment{{Name: "capture.png", MediaType: "image/png"}},
	)
	if err != nil {
		t.Fatalf("resolveImageSelection() error = %v", err)
	}
	if selection.ProviderID != "codex" || selection.ModelID != "model-fast" {
		t.Fatalf("resolved selection = %s/%s, want codex/model-fast", selection.ProviderID, selection.ModelID)
	}
	if selection.ReasoningEffort != "" {
		t.Fatalf("reasoning effort = %q, want unsupported effort cleared", selection.ReasoningEffort)
	}
	if !strings.Contains(classification.Reason, "direcionada ao Codex") {
		t.Fatalf("classification reason = %q, want explicit image routing reason", classification.Reason)
	}
}

func TestCanonicalContextDoesNotCopyImageData(t *testing.T) {
	imageData := base64.StdEncoding.EncodeToString([]byte("\x89PNG\r\n\x1a\n"))
	prompt, err := composeCanonicalContextPrompt([]Message{{
		Role: "user",
		Attachments: []Attachment{{
			Name: "capture.png", MediaType: "image/png", Content: imageData,
		}},
	}}, "Pedido atual")
	if err != nil {
		t.Fatalf("composeCanonicalContextPrompt() error = %v", err)
	}
	if strings.Contains(prompt, imageData) {
		t.Fatal("canonical context included image data")
	}
	if !strings.Contains(prompt, "[conteúdo visual não transferido entre providers]") {
		t.Fatal("canonical context did not explain that image content was omitted")
	}
}
