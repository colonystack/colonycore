package domain

import (
	"reflect"
	"testing"

	"colonycore/pkg/domain/entitymodel"
)

// Guard that domain entity wrappers continue to embed the generated shapes only.
// This prevents drift when the schema generator adds or removes fields.
func TestDomainEntitiesEmbedGeneratedModel(t *testing.T) {
	t.Helper()

	cases := []struct {
		name      string
		instance  any
		generated any
	}{
		{name: "Organism", instance: Organism{}, generated: entitymodel.Organism{}},
		{name: "Cohort", instance: Cohort{}, generated: entitymodel.Cohort{}},
		{name: "HousingUnit", instance: HousingUnit{}, generated: entitymodel.HousingUnit{}},
		{name: "Facility", instance: Facility{}, generated: entitymodel.Facility{}},
		{name: "BreedingUnit", instance: BreedingUnit{}, generated: entitymodel.BreedingUnit{}},
		{name: "Line", instance: Line{}, generated: entitymodel.Line{}},
		{name: "Strain", instance: Strain{}, generated: entitymodel.Strain{}},
		{name: "GenotypeMarker", instance: GenotypeMarker{}, generated: entitymodel.GenotypeMarker{}},
		{name: "Procedure", instance: Procedure{}, generated: entitymodel.Procedure{}},
		{name: "Treatment", instance: Treatment{}, generated: entitymodel.Treatment{}},
		{name: "Observation", instance: Observation{}, generated: entitymodel.Observation{}},
		{name: "Sample", instance: Sample{}, generated: entitymodel.Sample{}},
		{name: "Protocol", instance: Protocol{}, generated: entitymodel.Protocol{}},
		{name: "Permit", instance: Permit{}, generated: entitymodel.Permit{}},
		{name: "Project", instance: Project{}, generated: entitymodel.Project{}},
		{name: "SupplyItem", instance: SupplyItem{}, generated: entitymodel.SupplyItem{}},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			domainType := reflect.TypeOf(tc.instance)
			if domainType.Kind() != reflect.Struct {
				t.Fatalf("%s must be a struct, got %s", tc.name, domainType.Kind())
				return
			}
			genType := reflect.TypeOf(tc.generated)
			embedded := 0
			for i := 0; i < domainType.NumField(); i++ {
				field := domainType.Field(i)
				switch {
				case field.Anonymous && field.Type == genType:
					embedded++
				case field.IsExported() && !field.Anonymous:
					t.Fatalf("%s exposes unexpected field %q of type %s", tc.name, field.Name, field.Type)
				case field.Anonymous && field.IsExported():
					t.Fatalf("%s embeds unexpected exported field %q of type %s", tc.name, field.Name, field.Type)
				}
			}
			if embedded != 1 {
				t.Fatalf("%s must embed exactly one %s field, found %d", tc.name, genType, embedded)
			}
		})
	}
}
