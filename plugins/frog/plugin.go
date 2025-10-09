// Package frog implements a reference plugin providing frog-specific schema
// extensions, rules, and example dataset templates for demonstration purposes.
package frog

import (
	"context"
	"fmt"
	"strings"
	"time"

	"colonycore/pkg/datasetapi"
	"colonycore/pkg/pluginapi"
)

const (
	frogHabitatRuleName = "frog_habitat_warning"
	pluginName          = "frog"
)

// Plugin implements the frog reference module described in the RFC.
type Plugin struct{}

// New constructs a frog plugin instance.
func New() Plugin {
	return Plugin{}
}

// Name returns the plugin identifier.
func (Plugin) Name() string { return pluginName }

// Version returns the plugin semantic version.
func (Plugin) Version() string { return "0.1.0" }

// Register wires species-specific schema extensions and rules.
func (Plugin) Register(registry pluginapi.Registry) error {
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

	dialectProvider := datasetapi.GetDialectProvider()
	formatProvider := datasetapi.GetFormatProvider()

	if err := registry.RegisterDatasetTemplate(datasetapi.Template{
		Key:         "frog_population_snapshot",
		Version:     "0.1.0",
		Title:       "Frog Population Snapshot",
		Description: "Lists frog organisms with lifecycle, housing, and project context scoped to the caller's RBAC filters.",
		Dialect:     dialectProvider.DSL(),
		Query: `REPORT frog_population_snapshot
SELECT organism_id, organism_name, species, lifecycle_stage, project_id, protocol_id, housing_id, updated_at
FROM organisms
WHERE species ILIKE 'frog%'`,
		Parameters: []datasetapi.Parameter{
			{
				Name:        "stage",
				Type:        "string",
				Description: "Optional lifecycle stage filter using canonical stage identifiers.",
				Enum: func() []string {
					stages := datasetapi.NewLifecycleStageContext()
					return []string{
						stages.Planned().String(),
						stages.Larva().String(),
						stages.Juvenile().String(),
						stages.Adult().String(),
						stages.Retired().String(),
						stages.Deceased().String(),
					}
				}(),
			},
			{
				Name:        "as_of",
				Type:        "timestamp",
				Description: "Only include organisms updated on or before the provided RFC3339 timestamp.",
				Unit:        "iso8601",
			},
			{
				Name:        "include_retired",
				Type:        "boolean",
				Description: "Include retired frogs when no explicit stage filter is provided.",
				Default:     false,
			},
		},
		Columns: []datasetapi.Column{
			{Name: "organism_id", Type: "string", Description: "Primary identifier for the organism."},
			{Name: "organism_name", Type: "string", Description: "Common name or accession for the organism."},
			{Name: "species", Type: "string", Description: "Recorded species name."},
			{Name: "lifecycle_stage", Type: "string", Description: "Canonical lifecycle stage."},
			{Name: "project_id", Type: "string", Description: "Owning project identifier."},
			{Name: "protocol_id", Type: "string", Description: "Linked protocol identifier."},
			{Name: "housing_id", Type: "string", Description: "Housing assignment identifier."},
			{Name: "updated_at", Type: "timestamp", Unit: "iso8601", Description: "Timestamp of last organism update."},
		},
		Metadata: datasetapi.Metadata{
			Source:          "core.organisms",
			Documentation:   "docs/rfc/0001-colonycore-base-module.md#63-uiapi-composition",
			RefreshInterval: "PT15M",
			Tags:            []string{"frog", "population", "lifecycle"},
			Annotations: map[string]string{
				"unit_of_count":  "organism",
				"classification": "operational",
			},
		},
		OutputFormats: []datasetapi.Format{
			formatProvider.JSON(),
			formatProvider.CSV(),
			formatProvider.Parquet(),
			formatProvider.HTML(),
			formatProvider.PNG(),
		},
		Binder: frogPopulationBinder,
	}); err != nil {
		return err
	}
	return nil
}

type frogHabitatRule struct{}

func (frogHabitatRule) Name() string { return frogHabitatRuleName }

func (frogHabitatRule) Evaluate(_ context.Context, view pluginapi.RuleView, _ []pluginapi.Change) (pluginapi.Result, error) {
	var result pluginapi.Result

	for _, organism := range view.ListOrganisms() {
		specie := strings.ToLower(organism.Species())
		if !strings.Contains(specie, "frog") {
			continue
		}
		housingID, ok := organism.HousingID()
		if !ok {
			continue
		}
		housing, ok := view.FindHousingUnit(housingID)
		if !ok {
			continue
		}
		env := strings.ToLower(housing.Environment())
		if strings.Contains(env, "aquatic") || strings.Contains(env, "humid") {
			continue
		}

		entities := pluginapi.NewEntityContext()

		violation, err := pluginapi.NewViolationBuilder().
			WithRule(frogHabitatRuleName).
			WithMessage("frog assigned to non-aquatic/non-humid housing").
			WithEntity(entities.Organism()).
			WithEntityID(organism.ID()).
			BuildWarning()
		if err != nil {
			return pluginapi.Result{}, fmt.Errorf("failed to build violation: %w", err)
		}

		result = result.AddViolation(violation)
	}
	return result, nil
}

func frogPopulationBinder(env datasetapi.Environment) (datasetapi.Runner, error) {
	if env.Store == nil {
		return nil, fmt.Errorf("dataset environment missing store")
	}
	now := env.Now
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return func(ctx context.Context, req datasetapi.RunRequest) (datasetapi.RunResult, error) {
		var rows []datasetapi.Row
		stageFilter, _ := req.Parameters["stage"].(string)
		includeRetired, _ := req.Parameters["include_retired"].(bool)
		var asOfTime *time.Time
		if ts, ok := req.Parameters["as_of"].(time.Time); ok {
			t := ts
			asOfTime = &t
		}
		err := env.Store.View(ctx, func(view datasetapi.TransactionView) error {
			for _, organism := range view.ListOrganisms() {
				species := strings.ToLower(organism.Species())
				if !strings.Contains(species, "frog") {
					continue
				}
				if stageFilter != "" && organism.GetCurrentStage().String() != stageFilter {
					continue
				}
				if stageFilter == "" && !includeRetired && organism.IsRetired() {
					continue
				}
				if asOfTime != nil && organism.UpdatedAt().After(*asOfTime) {
					continue
				}
				if len(req.Scope.ProjectIDs) > 0 {
					projectID, ok := organism.ProjectID()
					if !ok || !contains(req.Scope.ProjectIDs, projectID) {
						continue
					}
				}
				if len(req.Scope.ProtocolIDs) > 0 {
					protocolID, ok := organism.ProtocolID()
					if !ok || !contains(req.Scope.ProtocolIDs, protocolID) {
						continue
					}
				}
				row := datasetapi.Row{
					"organism_id":     organism.ID(),
					"organism_name":   organism.Name(),
					"species":         organism.Species(),
					"lifecycle_stage": organism.GetCurrentStage().String(),
					"project_id":      valueOrNil(organism.ProjectID()),
					"protocol_id":     valueOrNil(organism.ProtocolID()),
					"housing_id":      valueOrNil(organism.HousingID()),
					"updated_at":      organism.UpdatedAt().UTC(),
				}
				rows = append(rows, row)
			}
			return nil
		})
		if err != nil {
			return datasetapi.RunResult{}, err
		}
		metadata := map[string]any{
			"row_count": len(rows),
			"source":    "core.organisms",
		}
		if stageFilter != "" {
			metadata["stage_filter"] = stageFilter
		}
		if len(req.Scope.ProjectIDs) > 0 {
			metadata["project_scope"] = req.Scope.ProjectIDs
		}
		if len(req.Scope.ProtocolIDs) > 0 {
			metadata["protocol_scope"] = req.Scope.ProtocolIDs
		}
		if asOfTime != nil {
			metadata["as_of"] = asOfTime.UTC()
		}
		formatProvider := datasetapi.GetFormatProvider()
		return datasetapi.RunResult{
			Schema:      req.Template.Columns,
			Rows:        rows,
			Metadata:    metadata,
			GeneratedAt: now(),
			Format:      formatProvider.JSON(),
		}, nil
	}, nil
}

func contains(list []string, target string) bool {
	for _, item := range list {
		if item == target {
			return true
		}
	}
	return false
}

func valueOrNil(value string, ok bool) any {
	if !ok {
		return nil
	}
	return value
}
