package datasetapi

const (
	datasetObservationShapeNarrative  = "narrative"
	datasetObservationShapeStructured = "structured"
	datasetObservationShapeMixed      = "mixed"
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
	return observationShapeRef{value: datasetObservationShapeNarrative}
}

func (observationShapeProvider) Structured() ObservationShapeRef {
	return observationShapeRef{value: datasetObservationShapeStructured}
}

func (observationShapeProvider) Mixed() ObservationShapeRef {
	return observationShapeRef{value: datasetObservationShapeMixed}
}

type observationShapeRef struct {
	value string
}

func (o observationShapeRef) String() string {
	return o.value
}

func (o observationShapeRef) HasStructuredPayload() bool {
	return o.value == datasetObservationShapeStructured || o.value == datasetObservationShapeMixed
}

func (o observationShapeRef) HasNarrativeNotes() bool {
	return o.value == datasetObservationShapeNarrative || o.value == datasetObservationShapeMixed
}

func (o observationShapeRef) Equals(other ObservationShapeRef) bool {
	if otherRef, ok := other.(observationShapeRef); ok {
		return o.value == otherRef.value
	}
	return false
}

func (o observationShapeRef) isObservationShapeRef() {}

// inferObservationShape derives a shape classification from structured data and notes presence.
func inferObservationShape(hasStructuredData, hasNotes bool) ObservationShapeRef {
	shapes := observationShapeProvider{}
	switch {
	case hasStructuredData && hasNotes:
		return shapes.Mixed()
	case hasStructuredData:
		return shapes.Structured()
	default:
		return shapes.Narrative()
	}
}
