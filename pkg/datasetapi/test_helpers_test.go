package datasetapi

func newCoreExtensionSet(h HookRef, payload map[string]any) ExtensionSet {
	if payload == nil {
		return NewExtensionSet(nil)
	}
	core := NewExtensionContributorContext().Core()
	return NewExtensionSet(map[string]map[string]map[string]any{
		h.value(): {
			core.value(): payload,
		},
	})
}
