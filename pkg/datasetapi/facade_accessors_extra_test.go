package datasetapi

import (
	"testing"
	"time"
)

func TestSupplyItemAccessors(t *testing.T) {
	now := time.Now().UTC()
	expires := now.Add(24 * time.Hour)
	hook := NewExtensionHookContext().SupplyItemAttributes()

	data := SupplyItemData{
		Base:           BaseData{ID: "supply", CreatedAt: now, UpdatedAt: now},
		SKU:            "SKU-1",
		Name:           "Supply",
		Description:    strPtr("desc"),
		QuantityOnHand: 2,
		Unit:           "box",
		LotNumber:      strPtr("LOT"),
		ExpiresAt:      &expires,
		FacilityIDs:    []string{"fac"},
		ProjectIDs:     []string{"proj"},
		ReorderLevel:   5,
		Extensions:     newCoreExtensionSet(hook, map[string]any{"color": "blue"}),
	}

	item := NewSupplyItem(data)

	if ids := item.FacilityIDs(); len(ids) != 1 || ids[0] != "fac" {
		t.Fatalf("facility ids mismatch: %+v", ids)
	}
	ids := item.ProjectIDs()
	if len(ids) != 1 || ids[0] != "proj" {
		t.Fatalf("project ids mismatch: %+v", ids)
	}
	if lvl := item.ReorderLevel(); lvl != 5 {
		t.Fatalf("expected reorder level 5, got %d", lvl)
	}
	if attr := item.Attributes(); attr["color"] != "blue" {
		t.Fatalf("expected attribute clone, got %+v", attr)
	} else {
		attr["color"] = "red"
	}
	if attr := item.Attributes(); attr["color"] != "blue" {
		t.Fatalf("expected attributes to be cloned, got %+v", attr)
	}

	if exp, ok := item.ExpiresAt(); !ok || exp == nil || !exp.Equal(expires) {
		t.Fatalf("expected expires at %v, got %v ok=%v", expires, exp, ok)
	}
	status := item.GetInventoryStatus(now)
	if !status.RequiresReorder() {
		t.Fatalf("expected inventory status to require reorder")
	}
	if !item.RequiresReorder(now) {
		t.Fatalf("expected RequiresReorder to be true")
	}
	if item.IsExpired(now) {
		t.Fatalf("expected supply not to be expired at current time")
	}
	if !item.IsExpired(expires.Add(48 * time.Hour)) {
		t.Fatalf("expected supply to be expired after expiration time")
	}
}

func TestPermitAccessors(t *testing.T) {
	now := time.Now().UTC()
	until := now.Add(48 * time.Hour)

	data := PermitData{
		Base:              BaseData{ID: "permit", CreatedAt: now, UpdatedAt: now},
		PermitNumber:      "PERM-1",
		Authority:         "Gov",
		Status:            datasetPermitStatusApproved,
		ValidFrom:         now,
		ValidUntil:        until,
		AllowedActivities: []string{"collect"},
		FacilityIDs:       []string{"fac"},
		ProtocolIDs:       []string{"prot"},
		Notes:             strPtr("note"),
	}

	permit := NewPermit(data)

	if ids := permit.FacilityIDs(); len(ids) != 1 || ids[0] != "fac" {
		t.Fatalf("facility ids mismatch: %+v", ids)
	}
	if ids := permit.ProtocolIDs(); len(ids) != 1 || ids[0] != "prot" {
		t.Fatalf("protocol ids mismatch: %+v", ids)
	}
	if vf := permit.ValidFrom(); !vf.Equal(now) {
		t.Fatalf("expected valid from %v, got %v", now, vf)
	}
	if vu := permit.ValidUntil(); !vu.Equal(until) {
		t.Fatalf("expected valid until %v, got %v", until, vu)
	}
	if permit.IsExpired(now) {
		t.Fatalf("expected permit active at current time")
	}
	if !permit.IsExpired(until.Add(24 * time.Hour)) {
		t.Fatalf("expected permit expired after valid until")
	}

	status := permit.GetStatus(now)
	if status.String() != datasetPermitStatusApproved || !status.IsActive() {
		t.Fatalf("expected approved status to be active, got %s", status.String())
	}
	if permit.GetStatus(until.Add(24*time.Hour)).String() != datasetPermitStatusExpired {
		t.Fatalf("expected approved permit to become expired after validity window")
	}

	onHold := NewPermit(PermitData{
		Base:              BaseData{ID: "permit-onhold", CreatedAt: now, UpdatedAt: now},
		PermitNumber:      "PERM-ONHOLD",
		Authority:         "Gov",
		Status:            datasetPermitStatusOnHold,
		AllowedActivities: []string{"collect"},
		FacilityIDs:       []string{"fac"},
		ProtocolIDs:       []string{"prot"},
	})
	if onHold.GetStatus(now).String() != datasetPermitStatusOnHold {
		t.Fatalf("expected on_hold status to remain on_hold")
	}

	archived := NewPermit(PermitData{
		Base:              BaseData{ID: "permit-archived", CreatedAt: now, UpdatedAt: now},
		PermitNumber:      "PERM-ARCH",
		Authority:         "Gov",
		Status:            datasetPermitStatusArchived,
		AllowedActivities: []string{"collect"},
		FacilityIDs:       []string{"fac"},
		ProtocolIDs:       []string{"prot"},
	})
	if !archived.GetStatus(now).IsArchived() {
		t.Fatalf("expected archived status to be archived")
	}
}
