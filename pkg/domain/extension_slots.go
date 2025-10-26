package domain

import "colonycore/pkg/domain/extension"

// EnsureDefaultAttributes returns the line default attributes slot, initialising
// it with the correct hook identifier when needed.
func (l *Line) EnsureDefaultAttributes() *extension.Slot {
	if l.DefaultAttributes == nil {
		l.DefaultAttributes = extension.NewSlot(extension.HookLineDefaultAttributes)
		return l.DefaultAttributes
	}
	_ = l.DefaultAttributes.BindHook(extension.HookLineDefaultAttributes)
	return l.DefaultAttributes
}

// EnsureExtensionOverrides returns the line extension overrides slot, ensuring
// it is bound to the correct hook.
func (l *Line) EnsureExtensionOverrides() *extension.Slot {
	if l.ExtensionOverrides == nil {
		l.ExtensionOverrides = extension.NewSlot(extension.HookLineExtensionOverrides)
		return l.ExtensionOverrides
	}
	_ = l.ExtensionOverrides.BindHook(extension.HookLineExtensionOverrides)
	return l.ExtensionOverrides
}

// EnsureAttributes returns the strain attributes slot, initialising it if necessary.
func (s *Strain) EnsureAttributes() *extension.Slot {
	if s.Attributes == nil {
		s.Attributes = extension.NewSlot(extension.HookStrainAttributes)
		return s.Attributes
	}
	_ = s.Attributes.BindHook(extension.HookStrainAttributes)
	return s.Attributes
}

// EnsureAttributes returns the genotype marker attributes slot, initialising it if necessary.
func (g *GenotypeMarker) EnsureAttributes() *extension.Slot {
	if g.Attributes == nil {
		g.Attributes = extension.NewSlot(extension.HookGenotypeMarkerAttributes)
		return g.Attributes
	}
	_ = g.Attributes.BindHook(extension.HookGenotypeMarkerAttributes)
	return g.Attributes
}
