package pluginapi

const (
	observationShapeNarrative  = "narrative"
	observationShapeStructured = "structured"
	observationShapeMixed      = "mixed"
)

// ObservationContext provides contextual access to observation data shape references.
type ObservationContext interface {
	Shapes() ObservationShapeProvider
}

// ObservationShapeProvider exposes canonical observation data shapes.
type ObservationShapeProvider interface {
	Narrative() ObservationShapeRef
	Structured() ObservationShapeRef
	Mixed() ObservationShapeRef
}

// ObservationShapeRef represents an opaque observation data shape reference.
type ObservationShapeRef interface {
	String() string
	HasStructuredPayload() bool
	HasNarrativeNotes() bool
	Equals(other ObservationShapeRef) bool
	isObservationShapeRef()
}

type observationContext struct{}

// NewObservationContext constructs the default observation context provider.
func NewObservationContext() ObservationContext {
	return observationContext{}
}

func (observationContext) Shapes() ObservationShapeProvider {
	return observationShapeProvider{}
}

type observationShapeProvider struct{}

func (observationShapeProvider) Narrative() ObservationShapeRef {
	return observationShapeRef{value: observationShapeNarrative}
}

func (observationShapeProvider) Structured() ObservationShapeRef {
	return observationShapeRef{value: observationShapeStructured}
}

func (observationShapeProvider) Mixed() ObservationShapeRef {
	return observationShapeRef{value: observationShapeMixed}
}

type observationShapeRef struct {
	value string
}

func (o observationShapeRef) String() string {
	return o.value
}

func (o observationShapeRef) HasStructuredPayload() bool {
	return o.value == observationShapeStructured || o.value == observationShapeMixed
}

func (o observationShapeRef) HasNarrativeNotes() bool {
	return o.value == observationShapeNarrative || o.value == observationShapeMixed
}

func (o observationShapeRef) Equals(other ObservationShapeRef) bool {
	if otherRef, ok := other.(observationShapeRef); ok {
		return o.value == otherRef.value
	}
	return false
}

func (o observationShapeRef) isObservationShapeRef() {}
