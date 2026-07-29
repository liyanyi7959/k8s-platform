package application

import (
	"sort"
	"strings"
	"time"
)

// ToolCatalog owns the in-memory representation of AI tool definitions. It
// deliberately has no dependency on handlers' infrastructure concerns, so an
// adapter can register database and Kubernetes backed handlers without owning
// catalog lookup, presentation, or diagnostic planning rules.
type ToolCatalog struct {
	definitions map[string]ToolDefinition
	order       []string
}

func NewToolCatalog() *ToolCatalog {
	return &ToolCatalog{
		definitions: make(map[string]ToolDefinition),
		order:       make([]string, 0, 12),
	}
}

// Register adds a definition or replaces an existing definition with the
// same normalized name while keeping its original insertion position.
func (c *ToolCatalog) Register(definition ToolDefinition) {
	if c == nil {
		return
	}
	name := strings.TrimSpace(definition.Name)
	if name == "" {
		return
	}
	if c.definitions == nil {
		c.definitions = make(map[string]ToolDefinition)
	}
	if _, exists := c.definitions[name]; !exists {
		c.order = append(c.order, name)
	}
	c.definitions[name] = definition
}

func (c *ToolCatalog) Get(name string) (ToolDefinition, bool) {
	if c == nil {
		return ToolDefinition{}, false
	}
	definition, ok := c.definitions[strings.TrimSpace(name)]
	return definition, ok
}

// List returns the registered definitions in their stable catalog display
// order: category, then name.
func (c *ToolCatalog) List() []ToolDefinition {
	if c == nil {
		return nil
	}
	definitions := make([]ToolDefinition, 0, len(c.definitions))
	for _, name := range c.order {
		if definition, ok := c.definitions[name]; ok {
			definitions = append(definitions, definition)
		}
	}
	sort.SliceStable(definitions, func(i, j int) bool {
		if definitions[i].Category == definitions[j].Category {
			return definitions[i].Name < definitions[j].Name
		}
		return definitions[i].Category < definitions[j].Category
	})
	return definitions
}

// ListCatalog projects tool definitions to the client-visible catalog and
// evaluates availability from the caller's permissions. Schema maps and
// permission slices are copied to keep catalog responses isolated from the
// registered handler definitions.
func (c *ToolCatalog) ListCatalog(userPermissions []string) []ToolCatalogItem {
	definitions := c.List()
	items := make([]ToolCatalogItem, 0, len(definitions))
	for _, definition := range definitions {
		missing := MissingToolPermissions(userPermissions, definition.RequiredPermissions)
		items = append(items, ToolCatalogItem{
			Name:                definition.Name,
			Category:            definition.Category,
			Description:         definition.Description,
			RequiredPermissions: append([]string(nil), definition.RequiredPermissions...),
			MissingPermissions:  missing,
			RiskLevel:           definition.RiskLevel,
			ConfirmLevel:        definition.ConfirmLevel,
			TimeoutSeconds:      int64(definition.Timeout / time.Second),
			RedactionPolicy:     definition.RedactionPolicy,
			InputSchema:         CloneToolJSONMap(definition.InputSchema),
			OutputSchema:        CloneToolJSONMap(definition.OutputSchema),
			Available:           len(missing) == 0,
		})
	}
	return items
}

// PlanAutoDiagnostics applies the AI diagnostic policy only to tools exposed
// by this catalog. The policy itself remains independent from tool handlers.
func (c *ToolCatalog) PlanAutoDiagnostics(request ToolContextRequest) []ToolPlanStep {
	if c == nil {
		return nil
	}
	return BuildAutoDiagnosticToolPlan(request, func(toolName string) bool {
		_, ok := c.Get(toolName)
		return ok
	})
}
