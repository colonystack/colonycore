package domain

import "colonycore/pkg/domain/extension"

// EnsureDefaultAttributes returns the line default attributes slot, initialising
// it with the correct hook identifier when needed.
func (l *Line) EnsureDefaultAttributes() *extension.Slot {
	if l.defaultAttributesSlot == nil {
		l.defaultAttributesSlot = slotFromContainer(extension.HookLineDefaultAttributes, l.ensureExtensionContainer())
	} else {
		_ = l.defaultAttributesSlot.BindHook(extension.HookLineDefaultAttributes)
	}
	return l.defaultAttributesSlot
}

// EnsureExtensionOverrides returns the line extension overrides slot, ensuring
// it is bound to the correct hook.
func (l *Line) EnsureExtensionOverrides() *extension.Slot {
	if l.extensionOverridesSlot == nil {
		l.extensionOverridesSlot = slotFromContainer(extension.HookLineExtensionOverrides, l.ensureExtensionContainer())
	} else {
		_ = l.extensionOverridesSlot.BindHook(extension.HookLineExtensionOverrides)
	}
	return l.extensionOverridesSlot
}

// EnsureAttributes returns the strain attributes slot, initialising it if necessary.
func (s *Strain) EnsureAttributes() *extension.Slot {
	if s.attributesSlot == nil {
		s.attributesSlot = slotFromContainer(extension.HookStrainAttributes, s.ensureExtensionContainer())
	} else {
		_ = s.attributesSlot.BindHook(extension.HookStrainAttributes)
	}
	return s.attributesSlot
}

// EnsureAttributes returns the genotype marker attributes slot, initialising it if necessary.
func (g *GenotypeMarker) EnsureAttributes() *extension.Slot {
	if g.attributesSlot == nil {
		g.attributesSlot = slotFromContainer(extension.HookGenotypeMarkerAttributes, g.ensureExtensionContainer())
	} else {
		_ = g.attributesSlot.BindHook(extension.HookGenotypeMarkerAttributes)
	}
	return g.attributesSlot
}
