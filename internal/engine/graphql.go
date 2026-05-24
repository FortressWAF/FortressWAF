package engine

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type GraphQLInspector struct {
	maxDepth           int
	maxCost            int
	maxAliases         int
	maxBatchSize       int
	maxTokens          int
	blockIntrospection bool
	blockSchema        bool
	allowedOps         map[string]bool
	restrictedFields   map[string]bool
	strictValidation   bool
}

func NewGraphQLInspector(cfg GraphQLConfig) *GraphQLInspector {
	allowed := make(map[string]bool)
	for _, op := range cfg.AllowedOperations {
		allowed[op] = true
	}

	restricted := make(map[string]bool)
	for _, f := range cfg.RestrictedFields {
		restricted[f] = true
	}

	return &GraphQLInspector{
		maxDepth:           cfg.MaxDepth,
		maxCost:            cfg.MaxCost,
		maxAliases:         cfg.MaxAliases,
		maxBatchSize:       cfg.MaxBatchSize,
		maxTokens:          cfg.MaxTokens,
		blockIntrospection: cfg.BlockIntrospection,
		blockSchema:        cfg.BlockSchema,
		allowedOps:         allowed,
		restrictedFields:   restricted,
		strictValidation:   cfg.StrictValidation,
	}
}

type GraphQLConfig struct {
	Enabled            bool
	MaxDepth           int
	MaxCost            int
	MaxAliases         int
	MaxBatchSize       int
	MaxTokens          int
	BlockIntrospection bool
	BlockSchema        bool
	AllowedOperations  []string
	RestrictedFields   []string
	StrictValidation   bool
}

func (g *GraphQLInspector) Name() string { return "graphql_protection" }

func (g *GraphQLInspector) Inspect(ctx *RequestContext) (*Decision, error) {
	if ctx.ContentType == "" || (!strings.Contains(ctx.ContentType, "graphql") && !strings.Contains(ctx.ContentType, "json")) {
		return &Decision{Action: ActionAllow}, nil
	}

	if ctx.Body == nil || len(ctx.Body) == 0 {
		return &Decision{Action: ActionAllow}, nil
	}

	if g.blockIntrospection && g.isIntrospectionQuery(ctx.Body) {
		return &Decision{
			Action:   ActionBlock,
			RuleID:   "GRAPHQL-001",
			RuleName: "GraphQL introspection blocked",
			Severity: "high",
			Score:    85,
			Evidence: "introspection query not allowed",
		}, nil
	}

	if g.blockSchema && g.isSchemaQuery(ctx.Body) {
		return &Decision{
			Action:   ActionBlock,
			RuleID:   "GRAPHQL-002",
			RuleName: "GraphQL schema query blocked",
			Severity: "high",
			Score:    85,
			Evidence: "schema query not allowed",
		}, nil
	}

	query, err := g.parseQuery(ctx.Body)
	if err != nil {
		if g.strictValidation {
			return &Decision{
				Action:   ActionBlock,
				RuleID:   "GRAPHQL-003",
				RuleName: "GraphQL parse error",
				Severity: "medium",
				Score:    60,
				Evidence: err.Error(),
			}, nil
		}
		return &Decision{Action: ActionAllow}, nil
	}

	if g.maxDepth > 0 && query.Depth > g.maxDepth {
		return &Decision{
			Action:   ActionBlock,
			RuleID:   "GRAPHQL-004",
			RuleName: "GraphQL depth limit exceeded",
			Severity: "high",
			Score:    80,
			Evidence: fmt.Sprintf("depth=%d, limit=%d", query.Depth, g.maxDepth),
		}, nil
	}

	if g.maxCost > 0 && query.Cost > g.maxCost {
		return &Decision{
			Action:   ActionBlock,
			RuleID:   "GRAPHQL-005",
			RuleName: "GraphQL cost limit exceeded",
			Severity: "high",
			Score:    80,
			Evidence: fmt.Sprintf("cost=%d, limit=%d", query.Cost, g.maxCost),
		}, nil
	}

	if g.maxAliases > 0 && query.Aliases > g.maxAliases {
		return &Decision{
			Action:   ActionBlock,
			RuleID:   "GRAPHQL-006",
			RuleName: "GraphQL alias limit exceeded",
			Severity: "medium",
			Score:    65,
			Evidence: fmt.Sprintf("aliases=%d, limit=%d", query.Aliases, g.maxAliases),
		}, nil
	}

	if g.maxBatchSize > 0 {
		batchSize := g.countOperations(ctx.Body)
		if batchSize > g.maxBatchSize {
			return &Decision{
				Action:   ActionBlock,
				RuleID:   "GRAPHQL-007",
				RuleName: "GraphQL batch size exceeded",
				Severity: "high",
				Score:    75,
				Evidence: fmt.Sprintf("batch_size=%d, limit=%d", batchSize, g.maxBatchSize),
			}, nil
		}
	}

	if len(g.allowedOps) > 0 && query.Operation != "" {
		if !g.allowedOps[query.Operation] {
			return &Decision{
				Action:   ActionBlock,
				RuleID:   "GRAPHQL-008",
				RuleName: "GraphQL operation not allowed",
				Severity: "high",
				Score:    80,
				Evidence: fmt.Sprintf("operation=%s", query.Operation),
			}, nil
		}
	}

	return &Decision{Action: ActionAllow}, nil
}

type GraphQLQuery struct {
	Operation string
	Depth     int
	Cost      int
	Aliases   int
}

func (g *GraphQLInspector) isIntrospectionQuery(body []byte) bool {
	bodyStr := string(body)
	introspectionPatterns := []string{
		"__schema",
		"__type",
		"IntrospectionQuery",
		"Introspection",
	}
	for _, pattern := range introspectionPatterns {
		if strings.Contains(bodyStr, pattern) {
			return true
		}
	}
	return false
}

func (g *GraphQLInspector) isSchemaQuery(body []byte) bool {
	bodyStr := strings.ToLower(string(body))
	return strings.Contains(bodyStr, "{ __typename }") && strings.Contains(bodyStr, "__type")
}

func (g *GraphQLInspector) parseQuery(body []byte) (*GraphQLQuery, error) {
	var parsed map[string]interface{}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("json unmarshal: %w", err)
	}

	query := &GraphQLQuery{}

	if v, ok := parsed["query"].(string); ok {
		g.analyzeQueryString(v, query)
	}

	if v, ok := parsed["operationName"].(string); ok {
		query.Operation = v
	}

	return query, nil
}

func (g *GraphQLInspector) analyzeQueryString(q string, query *GraphQLQuery) {
	trimmed := strings.TrimSpace(q)

	if strings.HasPrefix(trimmed, "query") {
		query.Operation = "query"
	} else if strings.HasPrefix(trimmed, "mutation") {
		query.Operation = "mutation"
	} else if strings.HasPrefix(trimmed, "subscription") {
		query.Operation = "subscription"
	} else {
		query.Operation = "query"
	}

	aliasRegex := regexp.MustCompile(`(\w+)\s*:`)
	aliases := aliasRegex.FindAllStringSubmatch(q, -1)
	query.Aliases = len(aliases)

	query.Depth = g.countDepth(q)
	query.Cost = g.calculateCost(q)
}

func (g *GraphQLInspector) countDepth(q string) int {
	maxDepth := 0
	currentDepth := 0

	lines := strings.Split(q, "\n")
	for _, line := range lines {
		for _, r := range line {
			if r == '{' {
				currentDepth++
				if currentDepth > maxDepth {
					maxDepth = currentDepth
				}
			} else if r == '}' {
				if currentDepth > 0 {
					currentDepth--
				}
			}
		}
	}

	return maxDepth
}

func (g *GraphQLInspector) calculateCost(q string) int {
	cost := 0

	patterns := []struct {
		pattern string
		cost    int
	}{
		{"query", 1},
		{"mutation", 10},
		{"subscription", 100},
		{"fragment", 5},
		{"... on", 20},
		{"@include", 2},
		{"@skip", 2},
		{"@deprecated", 1},
	}

	lower := strings.ToLower(q)
	for _, p := range patterns {
		cnt := strings.Count(lower, p.pattern)
		cost += cnt * p.cost
	}

	pageSize := g.countPageSize(q)
	cost += pageSize * pageSize

	return cost
}

func (g *GraphQLInspector) countPageSize(q string) int {
	pageSize := 100
	firstPos := strings.Index(strings.ToLower(q), "first:")
	if firstPos != -1 {
		endPos := firstPos + 6
		start := endPos
		for endPos < len(q) && q[endPos] >= '0' && q[endPos] <= '9' {
			endPos++
		}
		if endPos > start {
			if size, err := strconv.Atoi(q[start:endPos]); err == nil {
				pageSize = size
			}
		}
	}
	return pageSize
}

func (g *GraphQLInspector) countOperations(body []byte) int {
	if g.isJSONQuery(body) {
		var req struct {
			Operations []map[string]interface{} `json:"operations"`
		}
		if err := json.Unmarshal(body, &req); err == nil {
			return len(req.Operations)
		}

		var batchReq []map[string]interface{}
		if err := json.Unmarshal(body, &batchReq); err == nil {
			return len(batchReq)
		}
	}

	bodyStr := string(body)
	return strings.Count(bodyStr, `{"query"`)
}

func (g *GraphQLInspector) isJSONQuery(body []byte) bool {
	return bytes.HasPrefix(bytes.TrimSpace(body), []byte("{"))
}

func (g *GraphQLInspector) ValidateQuery(q string) (bool, string) {
	if g.maxTokens > 0 && len(q) > g.maxTokens {
		return false, fmt.Sprintf("query exceeds maximum tokens: %d > %d", len(q), g.maxTokens)
	}

	dangerousPatterns := []string{
		"${",
		"{{",
		"__dirname",
		"__filename",
		"process",
		"eval(",
		"require(",
		"import(",
	}
	lower := strings.ToLower(q)
	for _, pattern := range dangerousPatterns {
		if strings.Contains(lower, pattern) {
			return false, fmt.Sprintf("dangerous pattern detected: %s", pattern)
		}
	}

	return true, ""
}
