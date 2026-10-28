package personalagent

import (
	"path/filepath"
	"strings"
)

const (
	TaskClassQuick     = "quick"
	TaskClassCoding    = "coding"
	TaskClassResearch  = "research"
	TaskClassReasoning = "reasoning"
)

type TaskClassification struct {
	TaskClass string `json:"task_class"`
	Reason    string `json:"reason"`
}

type classificationSignal struct {
	class   string
	needles []string
	reason  string
}

var classificationSignals = []classificationSignal{
	{
		class: TaskClassCoding,
		needles: []string{
			"código", "codigo", "programar", "implementar", "refatorar", "depurar",
			"debug", "bug", "stack trace", "compilar", "build", "teste", "testes",
			"code", "coding", "implement", "refactor", "debug", "compile", "function",
		},
		reason: "o pedido menciona código, implementação ou depuração",
	},
	{
		class: TaskClassResearch,
		needles: []string{
			"pesquise", "pesquisar", "pesquisa", "fontes", "citações", "citacoes",
			"atual", "hoje", "último", "ultima", "últimas", "ultimas", "internet",
			"web", "navegue", "notícias", "noticias", "research", "latest", "current",
			"sources", "citations", "browse", "look up", "search for",
		},
		reason: "o pedido solicita pesquisa, fontes ou informação atual",
	},
	{
		class: TaskClassReasoning,
		needles: []string{
			"planeje", "planejar", "plano", "arquitetura", "compare", "comparar",
			"trade-off", "tradeoff", "decisão", "decisao", "estratégia", "estrategia",
			"análise", "analise", "raciocínio", "raciocinio", "explique", "explique por quê",
			"design", "plan", "architecture", "compare", "tradeoff", "strategy", "analyze",
			"reasoning", "explain", "evaluate", "deep dive",
		},
		reason: "o pedido pede planejamento, comparação ou análise em várias etapas",
	},
}

func ClassifyTask(content string, attachments []Attachment) TaskClassification {
	text := strings.ToLower(content)
	for _, attachment := range attachments {
		extension := strings.ToLower(filepath.Ext(attachment.Name))
		if isCodeAttachment(extension) {
			return TaskClassification{
				TaskClass: TaskClassCoding,
				Reason:    "o pedido inclui um arquivo de código",
			}
		}
	}
	for _, signal := range classificationSignals {
		for _, needle := range signal.needles {
			if strings.Contains(text, needle) {
				return TaskClassification{
					TaskClass: signal.class,
					Reason:    signal.reason + " (sinal: “" + needle + "”)",
				}
			}
		}
	}
	return TaskClassification{
		TaskClass: TaskClassQuick,
		Reason:    "nenhum sinal de pesquisa, código ou análise extensa foi encontrado",
	}
}

func isCodeAttachment(extension string) bool {
	switch extension {
	case ".c", ".cc", ".go", ".h", ".java", ".js", ".jsx", ".py", ".rs", ".sh", ".sql", ".ts", ".tsx":
		return true
	default:
		return false
	}
}

func routeRuleFor(config RoutingConfig, taskClass string) (RouteRule, bool) {
	for _, rule := range config.Rules {
		if rule.TaskClass == taskClass {
			return rule, true
		}
	}
	return RouteRule{}, false
}

func knownTaskClasses() []string {
	return []string{TaskClassQuick, TaskClassCoding, TaskClassResearch, TaskClassReasoning}
}
