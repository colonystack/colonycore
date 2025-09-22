package frog

import (
	"context"
	"strings"

	"colonycore/internal/core"
)

// Plugin implements the frog reference module described in the RFC (stubbed for the PoC).
type Plugin struct{}

// New constructs a frog plugin instance.
func New() Plugin {
	return Plugin{}
}

// Name returns the plugin identifier.
func (Plugin) Name() string { return "frog" }

// Version returns the plugin semantic version.
func (Plugin) Version() string { return "0.1.0" }

// Register wires species-specific schema extensions and rules.
func (Plugin) Register(registry *core.PluginRegistry) error {
	registry.RegisterSchema("organism", map[string]any{
		"$id":  "colonycore:frog:organism",
		"type": "object",
		"properties": map[string]any{
			"skin_color_index": map[string]any{
				"type":        "number",
				"minimum":     0,
				"maximum":     10,
				"description": "Fitzpatrick-inspired coloration scale",
			},
			"call_frequency_hz": map[string]any{
				"type":        "number",
				"description": "Dominant advertisement call frequency in Hz",
			},
			"limb_regeneration_notes": map[string]any{
				"type":        "string",
				"description": "Qualitative regeneration observations",
			},
		},
	})

	registry.RegisterRule(frogHabitatRule{})
	return nil
}

type frogHabitatRule struct{}

func (frogHabitatRule) Name() string { return "frog_habitat_warning" }

func (frogHabitatRule) Evaluate(ctx context.Context, view core.TransactionView, changes []core.Change) (core.Result, error) {
	var result core.Result
	for _, organism := range view.ListOrganisms() {
		specie := strings.ToLower(organism.Species)
		if !strings.Contains(specie, "frog") {
			continue
		}
		if organism.HousingID == nil {
			continue
		}
		housing, ok := view.FindHousingUnit(*organism.HousingID)
		if !ok {
			continue
		}
		env := strings.ToLower(housing.Environment)
		if strings.Contains(env, "aquatic") || strings.Contains(env, "humid") {
			continue
		}
		result.Violations = append(result.Violations, core.Violation{
			Rule:     "frog_habitat_warning",
			Severity: core.SeverityWarn,
			Message:  "frog assigned to non-aquatic/non-humid housing",
			Entity:   core.EntityOrganism,
			EntityID: organism.ID,
		})
	}
	return result, nil
}
