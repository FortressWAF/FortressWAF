package graphql

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/FortressWAF/FortressWAF/internal/engine"
)

type Inspector struct {
	mu                 sync.RWMutex
	maxDepth           int
	maxCost            int
	maxAliases         int
	maxBatchSize       int
	maxTokens          int
	blockIntrospection bool
	blockSchema        bool
	allowedOps         map[string]bool
	restrictedFields   map[string]bool
	queryCache         *queryCache
	depthLimit         int
	costLimit          int
	aliasLimit         int
	batchLimit         int
	strictValidation   bool
}

type Query struct {
	Operation string
	Name      string
	Variables map[string]interface{}
	Fragments map[string]string
	Depth     int
	Cost      int
	Aliases   int
}

type queryCache struct {
	mu    sync.RWMutex
	cache map[string]*CachedQuery
	ttl   int64
	size  int
}

type CachedQuery struct {
	Query    *Query
	ExpAt    int64
	HitCount int64
}

func New(cfg Config) *Inspector {
	allowed := make(map[string]bool)
	for _, op := range cfg.AllowedOperations {
		allowed[op] = true
	}

	restricted := make(map[string]bool)
	for _, f := range cfg.RestrictedFields {
		restricted[f] = true
	}

	return &Inspector{
		maxDepth:           cfg.MaxDepth,
		maxCost:            cfg.MaxCost,
		maxAliases:         cfg.MaxAliases,
		maxBatchSize:       cfg.MaxBatchSize,
		maxTokens:          cfg.MaxTokens,
		blockIntrospection: cfg.BlockIntrospection,
		blockSchema:        cfg.BlockSchema,
		allowedOps:         allowed,
		restrictedFields:   restricted,
		queryCache: &queryCache{
			cache: make(map[string]*CachedQuery),
			ttl:   300,
			size:  1000,
		},
		depthLimit:       cfg.MaxDepth,
		costLimit:        cfg.MaxCost,
		aliasLimit:       cfg.MaxAliases,
		batchLimit:       cfg.MaxBatchSize,
		strictValidation: cfg.StrictValidation,
	}
}

type Config struct {
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
	SchemaFile         string
}

func (g *Inspector) Name() string { return "graphql_protection" }

func (g *Inspector) Inspect(ctx *engine.RequestContext) (*engine.Decision, error) {
	if ctx.ContentType == "" || (!strings.Contains(ctx.ContentType, "graphql") && !strings.Contains(ctx.ContentType, "json")) {
		return &engine.Decision{Action: engine.ActionAllow}, nil
	}

	if ctx.Body == nil || len(ctx.Body) == 0 {
		return &engine.Decision{Action: engine.ActionAllow}, nil
	}

	if g.blockIntrospection && g.isIntrospectionQuery(ctx.Body) {
		return &engine.Decision{
			Action:   engine.ActionBlock,
			RuleID:   "GRAPHQL-001",
			RuleName: "GraphQL introspection blocked",
			Severity: "high",
			Score:    85,
			Evidence: "introspection query not allowed",
		}, nil
	}

	if g.blockSchema && g.isSchemaQuery(ctx.Body) {
		return &engine.Decision{
			Action:   engine.ActionBlock,
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
			return &engine.Decision{
				Action:   engine.ActionBlock,
				RuleID:   "GRAPHQL-003",
				RuleName: "GraphQL parse error",
				Severity: "medium",
				Score:    60,
				Evidence: err.Error(),
			}, nil
		}
		return &engine.Decision{Action: engine.ActionAllow}, nil
	}

	if g.depthLimit > 0 && query.Depth > g.depthLimit {
		return &engine.Decision{
			Action:   engine.ActionBlock,
			RuleID:   "GRAPHQL-004",
			RuleName: "GraphQL depth limit exceeded",
			Severity: "high",
			Score:    80,
			Evidence: fmt.Sprintf("depth=%d, limit=%d", query.Depth, g.depthLimit),
		}, nil
	}

	if g.costLimit > 0 && query.Cost > g.costLimit {
		return &engine.Decision{
			Action:   engine.ActionBlock,
			RuleID:   "GRAPHQL-005",
			RuleName: "GraphQL cost limit exceeded",
			Severity: "high",
			Score:    80,
			Evidence: fmt.Sprintf("cost=%d, limit=%d", query.Cost, g.costLimit),
		}, nil
	}

	if g.aliasLimit > 0 && query.Aliases > g.aliasLimit {
		return &engine.Decision{
			Action:   engine.ActionBlock,
			RuleID:   "GRAPHQL-006",
			RuleName: "GraphQL alias limit exceeded",
			Severity: "medium",
			Score:    65,
			Evidence: fmt.Sprintf("aliases=%d, limit=%d", query.Aliases, g.aliasLimit),
		}, nil
	}

	if g.batchLimit > 0 {
		batchSize := g.countOperations(ctx.Body)
		if batchSize > g.batchLimit {
			return &engine.Decision{
				Action:   engine.ActionBlock,
				RuleID:   "GRAPHQL-007",
				RuleName: "GraphQL batch size exceeded",
				Severity: "high",
				Score:    75,
				Evidence: fmt.Sprintf("batch_size=%d, limit=%d", batchSize, g.batchLimit),
			}, nil
		}
	}

	if len(g.allowedOps) > 0 && query.Operation != "" {
		if !g.allowedOps[query.Operation] {
			return &engine.Decision{
				Action:   engine.ActionBlock,
				RuleID:   "GRAPHQL-008",
				RuleName: "GraphQL operation not allowed",
				Severity: "high",
				Score:    80,
				Evidence: fmt.Sprintf("operation=%s", query.Operation),
			}, nil
		}
	}

	return &engine.Decision{Action: engine.ActionAllow}, nil
}

func (g *Inspector) isIntrospectionQuery(body []byte) bool {
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

func (g *Inspector) isSchemaQuery(body []byte) bool {
	bodyStr := strings.ToLower(string(body))
	return strings.Contains(bodyStr, "{ __typename }") && strings.Contains(bodyStr, "__type")
}

func (g *Inspector) parseQuery(body []byte) (*Query, error) {
	var parsed map[string]interface{}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("json unmarshal: %w", err)
	}

	query := &Query{
		Variables: make(map[string]interface{}),
		Fragments: make(map[string]string),
	}

	if v, ok := parsed["query"].(string); ok {
		g.analyzeQueryString(v, query)
	}

	if v, ok := parsed["operationName"].(string); ok {
		query.Name = v
	}

	if v, ok := parsed["variables"].(map[string]interface{}); ok {
		query.Variables = v
	}

	return query, nil
}

func (g *Inspector) analyzeQueryString(q string, query *Query) {
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

func (g *Inspector) countDepth(q string) int {
	maxDepth := 0
	currentDepth := 0

	lines := strings.Split(q, "\n")
	for _, line := range lines {
		for _, ch := range line {
			if ch == '{' {
				currentDepth++
				if currentDepth > maxDepth {
					maxDepth = currentDepth
				}
			} else if ch == '}' {
				if currentDepth > 0 {
					currentDepth--
				}
			}
		}
	}

	return maxDepth
}

func (g *Inspector) calculateCost(q string) int {
	cost := 0

	。武当派
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

func (g *Inspector) countPageSize(q string) int {
	pageSize := 100
	firstPos := strings.Index(strings.ToLower(q), "first:")
	if firstPos != -1 {
		endPos := firstPos + 6
		start := endPos
		for endPos < len(q) && (q[endPos] >= '0' && q[endPos] <= '9') {
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

func (g *Inspector) countOperations(body []byte) int {
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
	return strings.Count(bodyStr, "{\"query\"")
}

func (g *Inspector) isJSONQuery(body []byte) bool {
	return bytes.HasPrefix(bytes.TrimSpace(body), []byte("{"))
}

func (g *Inspector) ValidateQuery(q string) (bool, string) {
	if utf8.RuneCountInString(q) > g.maxTokens {
		return false, fmt.Sprintf("query exceeds maximum tokens: %d > %d", utf8.RuneCountInString(q), g.maxTokens)
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

func (g *Inspector) SetDepthLimit(n int) {
	g.mu.Lock()
	g.depthLimit = n
	g.mu.Unlock()
}

func (g *Inspector) SetCostLimit(n int) {
	g.mu.Lock()
	g.costLimit = n
	g.mu.Unlock()
}

var _ = bytes.Buffer
