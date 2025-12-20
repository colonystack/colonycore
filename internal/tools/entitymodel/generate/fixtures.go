package main

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
)

type fixtureSnapshot struct {
	Organisms    map[string]map[string]any `json:"organisms"`
	Cohorts      map[string]map[string]any `json:"cohorts"`
	Housing      map[string]map[string]any `json:"housing"`
	Facilities   map[string]map[string]any `json:"facilities"`
	Breeding     map[string]map[string]any `json:"breeding"`
	Lines        map[string]map[string]any `json:"lines"`
	Strains      map[string]map[string]any `json:"strains"`
	Markers      map[string]map[string]any `json:"markers"`
	Procedures   map[string]map[string]any `json:"procedures"`
	Treatments   map[string]map[string]any `json:"treatments"`
	Observations map[string]map[string]any `json:"observations"`
	Samples      map[string]map[string]any `json:"samples"`
	Protocols    map[string]map[string]any `json:"protocols"`
	Permits      map[string]map[string]any `json:"permits"`
	Projects     map[string]map[string]any `json:"projects"`
	Supplies     map[string]map[string]any `json:"supplies"`
}

// generateFixtures builds a canonical dataset covering every entity and relationship.
func generateFixtures(doc schemaDoc) ([]byte, error) {
	snapshot, err := buildFixtureSnapshot(doc)
	if err != nil {
		return nil, err
	}
	if err := validateFixture(snapshot, doc); err != nil {
		return nil, err
	}

	payload, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal fixtures: %w", err)
	}
	payload = append(payload, '\n')
	return payload, nil
}

func buildFixtureSnapshot(doc schemaDoc) (fixtureSnapshot, error) {
	if err := requireEnumValue(doc.Enums, "lifecycle_stage", "juvenile"); err != nil {
		return fixtureSnapshot{}, err
	}
	if err := requireEnumValue(doc.Enums, "housing_state", "active"); err != nil {
		return fixtureSnapshot{}, err
	}
	if err := requireEnumValue(doc.Enums, "housing_environment", "terrestrial"); err != nil {
		return fixtureSnapshot{}, err
	}
	if err := requireEnumValue(doc.Enums, "protocol_status", "approved"); err != nil {
		return fixtureSnapshot{}, err
	}
	if err := requireEnumValue(doc.Enums, "permit_status", "approved"); err != nil {
		return fixtureSnapshot{}, err
	}
	if err := requireEnumValue(doc.Enums, "procedure_status", "in_progress"); err != nil {
		return fixtureSnapshot{}, err
	}
	if err := requireEnumValue(doc.Enums, "treatment_status", "in_progress"); err != nil {
		return fixtureSnapshot{}, err
	}
	if err := requireEnumValue(doc.Enums, "sample_status", "stored"); err != nil {
		return fixtureSnapshot{}, err
	}

	const (
		baseTime      = "2025-01-01T00:00:00Z"
		scheduledTime = "2025-01-02T10:00:00Z"
		recordedTime  = "2025-01-03T12:00:00Z"
		collectionTS  = "2025-01-04T09:30:00Z"
		validUntil    = "2025-12-31T00:00:00Z"
	)

	facilityID := "00000000-0000-0000-0000-0000000000f1"
	housingID := "00000000-0000-0000-0000-0000000000h1"
	projectID := "00000000-0000-0000-0000-0000000000p1"
	protocolID := "00000000-0000-0000-0000-0000000000pr"
	permitID := "00000000-0000-0000-0000-0000000000pe"
	lineID := "00000000-0000-0000-0000-0000000000l1"
	strainID := "00000000-0000-0000-0000-0000000000s1"
	markerAID := "00000000-0000-0000-0000-0000000000m1"
	markerBID := "00000000-0000-0000-0000-0000000000m2"
	cohortID := "00000000-0000-0000-0000-0000000000c1"
	organismAID := "00000000-0000-0000-0000-0000000000o1"
	organismBID := "00000000-0000-0000-0000-0000000000o2"
	organismCID := "00000000-0000-0000-0000-0000000000o3"
	breedingID := "00000000-0000-0000-0000-0000000000b1"
	procedureID := "00000000-0000-0000-0000-0000000000r1"
	treatmentID := "00000000-0000-0000-0000-0000000000t1"
	observationID := "00000000-0000-0000-0000-0000000000ob"
	sampleID := "00000000-0000-0000-0000-0000000000sa"
	supplyID := "00000000-0000-0000-0000-0000000000su"

	lineLabel := "Fixture Line"

	snapshot := fixtureSnapshot{
		Facilities: map[string]map[string]any{
			facilityID: {
				"id":            facilityID,
				"created_at":    baseTime,
				"updated_at":    baseTime,
				"code":          "FAC-001",
				"name":          "Fixture Facility",
				"zone":          "Zone A",
				"access_policy": "standard",
				"environment_baselines": map[string]any{
					"core": map[string]any{
						"temperature_c": 24,
					},
				},
			},
		},
		Housing: map[string]map[string]any{
			housingID: {
				"id":          housingID,
				"created_at":  baseTime,
				"updated_at":  baseTime,
				"name":        "Main Habitat",
				"facility_id": facilityID,
				"capacity":    4,
				"environment": "terrestrial",
				"state":       "active",
			},
		},
		Projects: map[string]map[string]any{
			projectID: {
				"id":           projectID,
				"created_at":   baseTime,
				"updated_at":   baseTime,
				"code":         "PRJ-FXT",
				"title":        "Fixture Project",
				"description":  "Reference project for entity-model fixtures",
				"facility_ids": []string{facilityID},
				"protocol_ids": []string{protocolID},
			},
		},
		Protocols: map[string]map[string]any{
			protocolID: {
				"id":           protocolID,
				"created_at":   baseTime,
				"updated_at":   baseTime,
				"code":         "PROTO-FXT",
				"title":        "Fixture Protocol",
				"description":  "Approved protocol for fixture coverage",
				"max_subjects": 5,
				"status":       "approved",
			},
		},
		Permits: map[string]map[string]any{
			permitID: {
				"id":            permitID,
				"created_at":    baseTime,
				"updated_at":    baseTime,
				"permit_number": "PERMIT-123",
				"authority":     "Regulatory Body",
				"status":        "approved",
				"valid_from":    baseTime,
				"valid_until":   validUntil,
				"allowed_activities": []string{
					"housing",
					"sampling",
				},
				"facility_ids": []string{facilityID},
				"protocol_ids": []string{protocolID},
				"notes":        "Permit covering primary facility and protocol",
			},
		},
		Markers: map[string]map[string]any{
			markerAID: {
				"id":           markerAID,
				"created_at":   baseTime,
				"updated_at":   baseTime,
				"name":         "Marker A",
				"locus":        "LOC-A",
				"alleles":      []string{"A", "a"},
				"assay_method": "PCR",
				"interpretation": "Baseline genotype marker used for lineage " +
					"verification",
				"version": "v1",
				"attributes": map[string]any{
					"core": map[string]any{
						"expected_band": "120bp",
					},
				},
			},
			markerBID: {
				"id":           markerBID,
				"created_at":   baseTime,
				"updated_at":   baseTime,
				"name":         "Marker B",
				"locus":        "LOC-B",
				"alleles":      []string{"B", "b"},
				"assay_method": "PCR",
				"interpretation": "Secondary marker to exercise multiple marker " +
					"relationships",
				"version": "v1",
				"attributes": map[string]any{
					"core": map[string]any{
						"expected_band": "220bp",
					},
				},
			},
		},
		Lines: map[string]map[string]any{
			lineID: {
				"id":                  lineID,
				"created_at":          baseTime,
				"updated_at":          baseTime,
				"code":                "LINE-FXT",
				"name":                lineLabel,
				"origin":              "lab-maintained",
				"description":         "Reference line with genotype markers",
				"genotype_marker_ids": []string{markerAID, markerBID},
				"default_attributes": map[string]any{
					"core": map[string]any{
						"maintenance": "standard",
					},
				},
				"extension_overrides": map[string]any{
					"core": map[string]any{},
				},
			},
		},
		Strains: map[string]map[string]any{
			strainID: {
				"id":                  strainID,
				"created_at":          baseTime,
				"updated_at":          baseTime,
				"code":                "STR-FXT",
				"name":                "Fixture Strain",
				"line_id":             lineID,
				"description":         "Strain derived from fixture line",
				"generation":          "F1",
				"genotype_marker_ids": []string{markerAID},
			},
		},
		Cohorts: map[string]map[string]any{
			cohortID: {
				"id":          cohortID,
				"created_at":  baseTime,
				"updated_at":  baseTime,
				"name":        "Fixture Cohort",
				"purpose":     "Baseline study",
				"project_id":  projectID,
				"housing_id":  housingID,
				"protocol_id": protocolID,
			},
		},
		Organisms: map[string]map[string]any{
			organismAID: {
				"id":          organismAID,
				"created_at":  baseTime,
				"updated_at":  baseTime,
				"name":        "Alpha",
				"species":     "Specimenus fixture",
				"line":        lineLabel,
				"line_id":     lineID,
				"strain_id":   strainID,
				"stage":       "juvenile",
				"cohort_id":   cohortID,
				"housing_id":  housingID,
				"protocol_id": protocolID,
				"project_id":  projectID,
				"attributes": map[string]any{
					"core": map[string]any{
						"size_cm": 4.1,
					},
				},
			},
			organismBID: {
				"id":          organismBID,
				"created_at":  baseTime,
				"updated_at":  baseTime,
				"name":        "Bravo",
				"species":     "Specimenus fixture",
				"line":        lineLabel,
				"line_id":     lineID,
				"strain_id":   strainID,
				"stage":       "adult",
				"cohort_id":   cohortID,
				"housing_id":  housingID,
				"protocol_id": protocolID,
				"project_id":  projectID,
				"attributes": map[string]any{
					"core": map[string]any{
						"size_cm": 6.0,
					},
				},
			},
			organismCID: {
				"id":          organismCID,
				"created_at":  baseTime,
				"updated_at":  baseTime,
				"name":        "Charlie",
				"species":     "Specimenus fixture",
				"line":        lineLabel,
				"line_id":     lineID,
				"strain_id":   strainID,
				"stage":       "juvenile",
				"cohort_id":   cohortID,
				"housing_id":  housingID,
				"protocol_id": protocolID,
				"project_id":  projectID,
				"parent_ids":  []string{organismAID, organismBID},
				"attributes": map[string]any{
					"core": map[string]any{
						"size_cm": 3.2,
					},
				},
			},
		},
		Breeding: map[string]map[string]any{
			breedingID: {
				"id":               breedingID,
				"created_at":       baseTime,
				"updated_at":       baseTime,
				"name":             "Fixture Breeding Pair",
				"strategy":         "pairing",
				"housing_id":       housingID,
				"protocol_id":      protocolID,
				"line_id":          lineID,
				"strain_id":        strainID,
				"target_line_id":   lineID,
				"target_strain_id": strainID,
				"pairing_intent":   "Expand fixture line",
				"pairing_notes":    "Healthy adults selected",
				"female_ids":       []string{organismAID},
				"male_ids":         []string{organismBID},
				"pairing_attributes": map[string]any{
					"core": map[string]any{},
				},
			},
		},
		Procedures: map[string]map[string]any{
			procedureID: {
				"id":           procedureID,
				"created_at":   baseTime,
				"updated_at":   baseTime,
				"name":         "Fixture Procedure",
				"status":       "in_progress",
				"scheduled_at": scheduledTime,
				"protocol_id":  protocolID,
				"project_id":   projectID,
				"cohort_id":    cohortID,
				"organism_ids": []string{organismAID, organismBID},
			},
		},
		Treatments: map[string]map[string]any{
			treatmentID: {
				"id":           treatmentID,
				"created_at":   baseTime,
				"updated_at":   baseTime,
				"name":         "Fixture Treatment",
				"status":       "in_progress",
				"procedure_id": procedureID,
				"organism_ids": []string{organismAID, organismBID},
				"cohort_ids":   []string{cohortID},
				"dosage_plan":  "5ml oral dose",
				"administration_log": []string{
					"dose recorded at 10:00",
				},
				"adverse_events": []string{},
			},
		},
		Observations: map[string]map[string]any{
			observationID: {
				"id":           observationID,
				"created_at":   baseTime,
				"updated_at":   baseTime,
				"procedure_id": &procedureID,
				"organism_id":  &organismAID,
				"recorded_at":  recordedTime,
				"observer":     "observer@example.com",
				"notes":        "Stable vitals",
				"data": map[string]any{
					"core": map[string]any{
						"weight_g": 12.4,
					},
				},
			},
		},
		Samples: map[string]map[string]any{
			sampleID: {
				"id":               sampleID,
				"created_at":       baseTime,
				"updated_at":       baseTime,
				"identifier":       "SAMPLE-FXT-1",
				"source_type":      "organism",
				"organism_id":      organismAID,
				"facility_id":      facilityID,
				"collected_at":     collectionTS,
				"status":           "stored",
				"storage_location": "Freezer A1",
				"assay_type":       "PCR",
				"chain_of_custody": []map[string]any{
					{
						"actor":     "Technician One",
						"location":  "Lab Bench",
						"timestamp": collectionTS,
						"notes":     "Collected post procedure",
					},
				},
				"attributes": map[string]any{
					"core": map[string]any{
						"volume_ul": 120,
					},
				},
			},
		},
		Supplies: map[string]map[string]any{
			supplyID: {
				"id":               supplyID,
				"created_at":       baseTime,
				"updated_at":       baseTime,
				"sku":              "SUP-001",
				"name":             "Standard Feed",
				"description":      "Nutrient-rich feed for fixture organisms",
				"quantity_on_hand": 20,
				"unit":             "kg",
				"facility_ids":     []string{facilityID},
				"project_ids":      []string{projectID},
				"reorder_level":    5,
				"attributes": map[string]any{
					"core": map[string]any{
						"storage": "dry",
					},
				},
			},
		},
	}

	return snapshot, nil
}

func validateFixture(snapshot fixtureSnapshot, doc schemaDoc) error {
	for entity, spec := range doc.Entities {
		entries := snapshot.entities(entity)
		if len(entries) == 0 {
			return fmt.Errorf("fixture missing entity %s", entity)
		}
		for _, entry := range entries {
			for _, required := range spec.Required {
				if _, ok := entry[required]; !ok {
					return fmt.Errorf("fixture entity %s missing required field %s", entity, required)
				}
			}
			if err := ensureMinItems(entry, spec.Properties, entity); err != nil {
				return err
			}
		}
	}
	return nil
}

func ensureMinItems(entry map[string]any, rawProps map[string]json.RawMessage, entity string) error {
	for name, raw := range rawProps {
		var prop definitionSpec
		if err := json.Unmarshal(raw, &prop); err != nil {
			continue
		}
		if prop.Type != typeArray || prop.MinItems <= 0 {
			continue
		}
		val, ok := entry[name]
		if !ok {
			return fmt.Errorf("fixture entity %s missing required array %s", entity, name)
		}
		v := reflect.ValueOf(val)
		if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
			return fmt.Errorf("fixture entity %s field %s expected slice for minItems", entity, name)
		}
		if v.Len() < prop.MinItems {
			return fmt.Errorf("fixture entity %s field %s requires at least %d items", entity, name, prop.MinItems)
		}
	}
	return nil
}

func (f fixtureSnapshot) entities(name string) []map[string]any {
	switch name {
	case "Organism":
		return mapsFrom(f.Organisms)
	case "Cohort":
		return mapsFrom(f.Cohorts)
	case "HousingUnit":
		return mapsFrom(f.Housing)
	case "Facility":
		return mapsFrom(f.Facilities)
	case "BreedingUnit":
		return mapsFrom(f.Breeding)
	case "Line":
		return mapsFrom(f.Lines)
	case "Strain":
		return mapsFrom(f.Strains)
	case "GenotypeMarker":
		return mapsFrom(f.Markers)
	case "Procedure":
		return mapsFrom(f.Procedures)
	case "Treatment":
		return mapsFrom(f.Treatments)
	case "Observation":
		return mapsFrom(f.Observations)
	case "Sample":
		return mapsFrom(f.Samples)
	case "Protocol":
		return mapsFrom(f.Protocols)
	case "Permit":
		return mapsFrom(f.Permits)
	case "Project":
		return mapsFrom(f.Projects)
	case "SupplyItem":
		return mapsFrom(f.Supplies)
	default:
		return nil
	}
}

func mapsFrom(values map[string]map[string]any) []map[string]any {
	if len(values) == 0 {
		return nil
	}
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]map[string]any, 0, len(keys))
	for _, k := range keys {
		out = append(out, values[k])
	}
	return out
}

func requireEnumValue(enums map[string]enumSpec, name, value string) error {
	enum, ok := enums[name]
	if !ok {
		return fmt.Errorf("schema missing enum %s", name)
	}
	for _, candidate := range enum.Values {
		if candidate == value {
			return nil
		}
	}
	return fmt.Errorf("enum %s missing required value %s", name, value)
}
