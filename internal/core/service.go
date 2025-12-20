package core

import (
	"colonycore/internal/entitymodel"
	"colonycore/pkg/datasetapi"
	"colonycore/pkg/domain"
	"colonycore/pkg/pluginapi"
	"context"
	"fmt"
	"sync"
	"time"
)

// Clock exposes time retrieval used by the service for deterministic binding.
type Clock interface {
	Now() time.Time
}

// ClockFunc adapts a function into a Clock.
type ClockFunc func() time.Time

// Now returns the current time for the function-based clock.
func (fn ClockFunc) Now() time.Time {
	if fn == nil {
		return time.Now().UTC()
	}
	return fn().UTC()
}

// Logger abstracts structured logging used by the service layer.
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

type noopLogger struct{}

func (noopLogger) Debug(string, ...any) {}
func (noopLogger) Info(string, ...any)  {}
func (noopLogger) Warn(string, ...any)  {}
func (noopLogger) Error(string, ...any) {}

// AuditStatus describes the outcome of a service operation for audit trail entries.
type AuditStatus string

const (
	// AuditStatusSuccess indicates the operation completed without error.
	AuditStatusSuccess AuditStatus = "success"
	// AuditStatusError indicates the operation failed.
	AuditStatusError AuditStatus = "error"
)

// AuditEntry captures structured audit metadata for service operations.
type AuditEntry struct {
	Operation string
	Entity    domain.EntityType
	Action    domain.Action
	EntityID  string
	Status    AuditStatus
	Error     string
	Duration  time.Duration
	Timestamp time.Time
}

// AuditRecorder records audit entries emitted by service operations.
type AuditRecorder interface {
	Record(ctx context.Context, entry AuditEntry)
}

// MetricsRecorder observes operation timings and success results.
type MetricsRecorder interface {
	Observe(ctx context.Context, operation string, success bool, duration time.Duration)
}

// TraceSpan represents an in-flight tracing span.
type TraceSpan interface {
	End(err error)
}

// Tracer starts tracing spans for service operations.
type Tracer interface {
	Start(ctx context.Context, operation string) (context.Context, TraceSpan)
}

type noopAuditRecorder struct{}

func (noopAuditRecorder) Record(context.Context, AuditEntry) {}

type noopMetricsRecorder struct{}

func (noopMetricsRecorder) Observe(context.Context, string, bool, time.Duration) {}

type noopTracer struct{}

func (noopTracer) Start(ctx context.Context, _ string) (context.Context, TraceSpan) {
	return ctx, noopSpan{}
}

type noopSpan struct{}

func (noopSpan) End(error) {}

// ServiceOption configures optional dependencies for the Service constructor.
type ServiceOption func(*serviceOptions)

type serviceOptions struct {
	clock   Clock
	logger  Logger
	audit   AuditRecorder
	metrics MetricsRecorder
	tracer  Tracer
}

// WithClock overrides the default clock used by the service.
func WithClock(clock Clock) ServiceOption {
	return func(opts *serviceOptions) {
		if clock != nil {
			opts.clock = clock
		}
	}
}

func entityModelMajorFromPlugin(plugin pluginapi.Plugin) (int, bool) {
	if provider, ok := plugin.(pluginapi.EntityModelCompatibilityProvider); ok {
		major := provider.EntityModelMajor()
		if major < 0 {
			return 0, false
		}
		return major, true
	}
	return 0, false
}

func entityModelMajorFromTemplate(template datasetapi.Template) (int, bool) {
	if template.Metadata.EntityModelMajor == nil {
		return 0, false
	}
	major := *template.Metadata.EntityModelMajor
	if major < 0 {
		return 0, false
	}
	return major, true
}

func requireEntityModelCompatibility(expected int, source string) error {
	if expected < 0 {
		return nil
	}
	hostMajor, ok := entitymodel.MajorVersion()
	if !ok {
		return fmt.Errorf("entity model version unavailable; %s requires major %d", source, expected)
	}
	if hostMajor == expected {
		return nil
	}
	return fmt.Errorf("entity model major mismatch: host=%d, %s requires %d", hostMajor, source, expected)
}

func ensureTemplateCompatibility(templateMajor int, templateDeclared bool, pluginMajor int, pluginDeclared bool, slug string) error {
	if templateDeclared && pluginDeclared && templateMajor != pluginMajor {
		return fmt.Errorf("dataset template %s declares entity model major %d but plugin declares %d", slug, templateMajor, pluginMajor)
	}
	return nil
}

// WithLogger injects a logger used by the service.
func WithLogger(logger Logger) ServiceOption {
	return func(opts *serviceOptions) {
		if logger != nil {
			opts.logger = logger
		}
	}
}

// WithAuditRecorder injects an audit recorder used to track service operations.
func WithAuditRecorder(recorder AuditRecorder) ServiceOption {
	return func(opts *serviceOptions) {
		if recorder != nil {
			opts.audit = recorder
		}
	}
}

// WithMetricsRecorder injects a metrics recorder used to observe operation timings.
func WithMetricsRecorder(recorder MetricsRecorder) ServiceOption {
	return func(opts *serviceOptions) {
		if recorder != nil {
			opts.metrics = recorder
		}
	}
}

// WithTracer injects a tracer used to create spans for service operations.
func WithTracer(tracer Tracer) ServiceOption {
	return func(opts *serviceOptions) {
		if tracer != nil {
			opts.tracer = tracer
		}
	}
}

func defaultServiceOptions() serviceOptions {
	return serviceOptions{
		clock:   ClockFunc(func() time.Time { return time.Now().UTC() }),
		logger:  noopLogger{},
		audit:   noopAuditRecorder{},
		metrics: noopMetricsRecorder{},
		tracer:  noopTracer{},
	}
}

// Service orchestrates transactional operations, plugin registration, and dataset binding.
type Service struct {
	store    domain.PersistentStore
	engine   *domain.RulesEngine
	clock    Clock
	now      func() time.Time
	logger   Logger
	audit    AuditRecorder
	metrics  MetricsRecorder
	tracer   Tracer
	plugins  map[string]PluginMetadata
	datasets map[string]DatasetTemplate
	mu       sync.RWMutex
}

// NewService constructs a service backed by the supplied store.
func NewService(store domain.PersistentStore, opts ...ServiceOption) *Service {
	if store == nil {
		panic("core: service requires a persistent store")
	}
	options := defaultServiceOptions()
	for _, opt := range opts {
		if opt != nil {
			opt(&options)
		}
	}
	svc := &Service{
		store:    store,
		clock:    options.clock,
		logger:   options.logger,
		audit:    options.audit,
		metrics:  options.metrics,
		tracer:   options.tracer,
		plugins:  make(map[string]PluginMetadata),
		datasets: make(map[string]DatasetTemplate),
	}
	svc.engine = extractRulesEngine(store)
	svc.now = selectNowFunc(store, svc.clock)
	return svc
}

// NewInMemoryService creates a service and in-memory store with the given rules engine.
func NewInMemoryService(engine *domain.RulesEngine, opts ...ServiceOption) *Service {
	store := NewMemoryStore(engine)
	return NewService(store, opts...)
}

// Store returns the underlying storage implementation.
func (s *Service) Store() domain.PersistentStore {
	return s.store
}

// CreateProject persists a new project.
func (s *Service) CreateProject(ctx context.Context, project domain.Project) (domain.Project, domain.Result, error) {
	var created domain.Project
	res, dur, err := s.run(ctx, "create_project", func(tx domain.Transaction) error {
		var innerErr error
		created, innerErr = tx.CreateProject(project)
		return innerErr
	})
	if err == nil {
		s.recordAuditSuccess(ctx, "create_project", created.ID, dur)
	}
	return created, res, err
}

// UpdateProject mutates a project.
func (s *Service) UpdateProject(ctx context.Context, id string, mutator func(*domain.Project) error) (domain.Project, domain.Result, error) {
	var updated domain.Project
	res, dur, err := s.run(ctx, "update_project", func(tx domain.Transaction) error {
		var innerErr error
		updated, innerErr = tx.UpdateProject(id, mutator)
		return innerErr
	})
	if err == nil {
		s.recordAuditSuccess(ctx, "update_project", updated.ID, dur)
	}
	return updated, res, err
}

// DeleteProject removes a project.
func (s *Service) DeleteProject(ctx context.Context, id string) (domain.Result, error) {
	res, dur, err := s.run(ctx, "delete_project", func(tx domain.Transaction) error {
		return tx.DeleteProject(id)
	})
	if err == nil {
		s.recordAuditSuccess(ctx, "delete_project", id, dur)
	}
	return res, err
}

// CreateProtocol persists a new protocol.
func (s *Service) CreateProtocol(ctx context.Context, protocol domain.Protocol) (domain.Protocol, domain.Result, error) {
	var created domain.Protocol
	res, dur, err := s.run(ctx, "create_protocol", func(tx domain.Transaction) error {
		var innerErr error
		created, innerErr = tx.CreateProtocol(protocol)
		return innerErr
	})
	if err == nil {
		s.recordAuditSuccess(ctx, "create_protocol", created.ID, dur)
	}
	return created, res, err
}

// UpdateProtocol mutates a protocol.
func (s *Service) UpdateProtocol(ctx context.Context, id string, mutator func(*domain.Protocol) error) (domain.Protocol, domain.Result, error) {
	var updated domain.Protocol
	res, dur, err := s.run(ctx, "update_protocol", func(tx domain.Transaction) error {
		var innerErr error
		updated, innerErr = tx.UpdateProtocol(id, mutator)
		return innerErr
	})
	if err == nil {
		s.recordAuditSuccess(ctx, "update_protocol", updated.ID, dur)
	}
	return updated, res, err
}

// DeleteProtocol removes a protocol.
func (s *Service) DeleteProtocol(ctx context.Context, id string) (domain.Result, error) {
	res, dur, err := s.run(ctx, "delete_protocol", func(tx domain.Transaction) error {
		return tx.DeleteProtocol(id)
	})
	if err == nil {
		s.recordAuditSuccess(ctx, "delete_protocol", id, dur)
	}
	return res, err
}

// CreateFacility persists facility metadata.
func (s *Service) CreateFacility(ctx context.Context, facility domain.Facility) (domain.Facility, domain.Result, error) {
	var created domain.Facility
	res, dur, err := s.run(ctx, "create_facility", func(tx domain.Transaction) error {
		var innerErr error
		created, innerErr = tx.CreateFacility(facility)
		return innerErr
	})
	if err == nil {
		s.recordAuditSuccess(ctx, "create_facility", created.ID, dur)
	}
	return created, res, err
}

// UpdateFacility mutates a facility record.
func (s *Service) UpdateFacility(ctx context.Context, id string, mutator func(*domain.Facility) error) (domain.Facility, domain.Result, error) {
	var updated domain.Facility
	res, dur, err := s.run(ctx, "update_facility", func(tx domain.Transaction) error {
		var innerErr error
		updated, innerErr = tx.UpdateFacility(id, mutator)
		return innerErr
	})
	if err == nil {
		s.recordAuditSuccess(ctx, "update_facility", updated.ID, dur)
	}
	return updated, res, err
}

// DeleteFacility removes a facility.
func (s *Service) DeleteFacility(ctx context.Context, id string) (domain.Result, error) {
	res, dur, err := s.run(ctx, "delete_facility", func(tx domain.Transaction) error {
		return tx.DeleteFacility(id)
	})
	if err == nil {
		s.recordAuditSuccess(ctx, "delete_facility", id, dur)
	}
	return res, err
}

// CreateHousingUnit persists housing metadata.
func (s *Service) CreateHousingUnit(ctx context.Context, housing domain.HousingUnit) (domain.HousingUnit, domain.Result, error) {
	var created domain.HousingUnit
	res, dur, err := s.run(ctx, "create_housing_unit", func(tx domain.Transaction) error {
		var innerErr error
		created, innerErr = tx.CreateHousingUnit(housing)
		return innerErr
	})
	if err == nil {
		s.recordAuditSuccess(ctx, "create_housing_unit", created.ID, dur)
	}
	return created, res, err
}

// UpdateHousingUnit mutates a housing unit.
func (s *Service) UpdateHousingUnit(ctx context.Context, id string, mutator func(*domain.HousingUnit) error) (domain.HousingUnit, domain.Result, error) {
	var updated domain.HousingUnit
	res, dur, err := s.run(ctx, "update_housing_unit", func(tx domain.Transaction) error {
		var innerErr error
		updated, innerErr = tx.UpdateHousingUnit(id, mutator)
		return innerErr
	})
	if err == nil {
		s.recordAuditSuccess(ctx, "update_housing_unit", updated.ID, dur)
	}
	return updated, res, err
}

// DeleteHousingUnit removes a housing unit.
func (s *Service) DeleteHousingUnit(ctx context.Context, id string) (domain.Result, error) {
	res, dur, err := s.run(ctx, "delete_housing_unit", func(tx domain.Transaction) error {
		return tx.DeleteHousingUnit(id)
	})
	if err == nil {
		s.recordAuditSuccess(ctx, "delete_housing_unit", id, dur)
	}
	return res, err
}

// CreateCohort persists a new cohort.
func (s *Service) CreateCohort(ctx context.Context, cohort domain.Cohort) (domain.Cohort, domain.Result, error) {
	var created domain.Cohort
	res, dur, err := s.run(ctx, "create_cohort", func(tx domain.Transaction) error {
		var innerErr error
		created, innerErr = tx.CreateCohort(cohort)
		return innerErr
	})
	if err == nil {
		s.recordAuditSuccess(ctx, "create_cohort", created.ID, dur)
	}
	return created, res, err
}

// CreateOrganism persists a new organism.
func (s *Service) CreateOrganism(ctx context.Context, organism domain.Organism) (domain.Organism, domain.Result, error) {
	var created domain.Organism
	res, dur, err := s.run(ctx, "create_organism", func(tx domain.Transaction) error {
		var innerErr error
		created, innerErr = tx.CreateOrganism(organism)
		return innerErr
	})
	if err == nil {
		s.recordAuditSuccess(ctx, "create_organism", created.ID, dur)
	}
	return created, res, err
}

// UpdateOrganism mutates an organism using the provided mutator.
func (s *Service) UpdateOrganism(ctx context.Context, id string, mutator func(*domain.Organism) error) (domain.Organism, domain.Result, error) {
	var updated domain.Organism
	res, dur, err := s.run(ctx, "update_organism", func(tx domain.Transaction) error {
		var innerErr error
		updated, innerErr = tx.UpdateOrganism(id, mutator)
		return innerErr
	})
	if err == nil {
		s.recordAuditSuccess(ctx, "update_organism", updated.ID, dur)
	}
	return updated, res, err
}

// DeleteOrganism removes an organism record.
func (s *Service) DeleteOrganism(ctx context.Context, id string) (domain.Result, error) {
	res, dur, err := s.run(ctx, "delete_organism", func(tx domain.Transaction) error {
		return tx.DeleteOrganism(id)
	})
	if err == nil {
		s.recordAuditSuccess(ctx, "delete_organism", id, dur)
	}
	return res, err
}

// AssignOrganismHousing updates an organism's housing reference within a transaction that validates dependencies.
func (s *Service) AssignOrganismHousing(ctx context.Context, organismID, housingID string) (domain.Organism, domain.Result, error) {
	var updated domain.Organism
	res, dur, err := s.run(ctx, "assign_organism_housing", func(tx domain.Transaction) error {
		if _, ok := tx.FindHousingUnit(housingID); !ok {
			return ErrNotFound{Entity: domain.EntityHousingUnit, ID: housingID}
		}
		var innerErr error
		updated, innerErr = tx.UpdateOrganism(organismID, func(o *domain.Organism) error {
			o.HousingID = &housingID
			return nil
		})
		return innerErr
	})
	if err == nil {
		s.recordAuditSuccess(ctx, "assign_organism_housing", updated.ID, dur)
	}
	return updated, res, err
}

// AssignOrganismProtocol links an organism to a protocol within the same transactional scope.
func (s *Service) AssignOrganismProtocol(ctx context.Context, organismID, protocolID string) (domain.Organism, domain.Result, error) {
	var updated domain.Organism
	res, dur, err := s.run(ctx, "assign_organism_protocol", func(tx domain.Transaction) error {
		if _, ok := tx.FindProtocol(protocolID); !ok {
			return ErrNotFound{Entity: domain.EntityProtocol, ID: protocolID}
		}
		var innerErr error
		updated, innerErr = tx.UpdateOrganism(organismID, func(o *domain.Organism) error {
			o.ProtocolID = &protocolID
			return nil
		})
		return innerErr
	})
	if err == nil {
		s.recordAuditSuccess(ctx, "assign_organism_protocol", updated.ID, dur)
	}
	return updated, res, err
}

// CreateBreedingUnit persists a breeding configuration.
func (s *Service) CreateBreedingUnit(ctx context.Context, unit domain.BreedingUnit) (domain.BreedingUnit, domain.Result, error) {
	var created domain.BreedingUnit
	res, dur, err := s.run(ctx, "create_breeding_unit", func(tx domain.Transaction) error {
		var innerErr error
		created, innerErr = tx.CreateBreedingUnit(unit)
		return innerErr
	})
	if err == nil {
		s.recordAuditSuccess(ctx, "create_breeding_unit", created.ID, dur)
	}
	return created, res, err
}

// CreateProcedure persists a procedure record.
func (s *Service) CreateProcedure(ctx context.Context, procedure domain.Procedure) (domain.Procedure, domain.Result, error) {
	var created domain.Procedure
	res, dur, err := s.run(ctx, "create_procedure", func(tx domain.Transaction) error {
		var innerErr error
		created, innerErr = tx.CreateProcedure(procedure)
		return innerErr
	})
	if err == nil {
		s.recordAuditSuccess(ctx, "create_procedure", created.ID, dur)
	}
	return created, res, err
}

// UpdateProcedure mutates a procedure.
func (s *Service) UpdateProcedure(ctx context.Context, id string, mutator func(*domain.Procedure) error) (domain.Procedure, domain.Result, error) {
	var updated domain.Procedure
	res, dur, err := s.run(ctx, "update_procedure", func(tx domain.Transaction) error {
		var innerErr error
		updated, innerErr = tx.UpdateProcedure(id, mutator)
		return innerErr
	})
	if err == nil {
		s.recordAuditSuccess(ctx, "update_procedure", updated.ID, dur)
	}
	return updated, res, err
}

// DeleteProcedure removes a procedure record.
func (s *Service) DeleteProcedure(ctx context.Context, id string) (domain.Result, error) {
	res, dur, err := s.run(ctx, "delete_procedure", func(tx domain.Transaction) error {
		return tx.DeleteProcedure(id)
	})
	if err == nil {
		s.recordAuditSuccess(ctx, "delete_procedure", id, dur)
	}
	return res, err
}

// CreateTreatment persists a treatment record.
func (s *Service) CreateTreatment(ctx context.Context, treatment domain.Treatment) (domain.Treatment, domain.Result, error) {
	var created domain.Treatment
	res, dur, err := s.run(ctx, "create_treatment", func(tx domain.Transaction) error {
		var innerErr error
		created, innerErr = tx.CreateTreatment(treatment)
		return innerErr
	})
	if err == nil {
		s.recordAuditSuccess(ctx, "create_treatment", created.ID, dur)
	}
	return created, res, err
}

// UpdateTreatment mutates a treatment record.
func (s *Service) UpdateTreatment(ctx context.Context, id string, mutator func(*domain.Treatment) error) (domain.Treatment, domain.Result, error) {
	var updated domain.Treatment
	res, dur, err := s.run(ctx, "update_treatment", func(tx domain.Transaction) error {
		var innerErr error
		updated, innerErr = tx.UpdateTreatment(id, mutator)
		return innerErr
	})
	if err == nil {
		s.recordAuditSuccess(ctx, "update_treatment", updated.ID, dur)
	}
	return updated, res, err
}

// DeleteTreatment removes a treatment.
func (s *Service) DeleteTreatment(ctx context.Context, id string) (domain.Result, error) {
	res, dur, err := s.run(ctx, "delete_treatment", func(tx domain.Transaction) error {
		return tx.DeleteTreatment(id)
	})
	if err == nil {
		s.recordAuditSuccess(ctx, "delete_treatment", id, dur)
	}
	return res, err
}

// CreateObservation persists an observation.
func (s *Service) CreateObservation(ctx context.Context, observation domain.Observation) (domain.Observation, domain.Result, error) {
	var created domain.Observation
	res, dur, err := s.run(ctx, "create_observation", func(tx domain.Transaction) error {
		var innerErr error
		created, innerErr = tx.CreateObservation(observation)
		return innerErr
	})
	if err == nil {
		s.recordAuditSuccess(ctx, "create_observation", created.ID, dur)
	}
	return created, res, err
}

// UpdateObservation mutates an observation.
func (s *Service) UpdateObservation(ctx context.Context, id string, mutator func(*domain.Observation) error) (domain.Observation, domain.Result, error) {
	var updated domain.Observation
	res, dur, err := s.run(ctx, "update_observation", func(tx domain.Transaction) error {
		var innerErr error
		updated, innerErr = tx.UpdateObservation(id, mutator)
		return innerErr
	})
	if err == nil {
		s.recordAuditSuccess(ctx, "update_observation", updated.ID, dur)
	}
	return updated, res, err
}

// DeleteObservation removes an observation.
func (s *Service) DeleteObservation(ctx context.Context, id string) (domain.Result, error) {
	res, dur, err := s.run(ctx, "delete_observation", func(tx domain.Transaction) error {
		return tx.DeleteObservation(id)
	})
	if err == nil {
		s.recordAuditSuccess(ctx, "delete_observation", id, dur)
	}
	return res, err
}

// CreateSample persists a sample.
func (s *Service) CreateSample(ctx context.Context, sample domain.Sample) (domain.Sample, domain.Result, error) {
	var created domain.Sample
	res, dur, err := s.run(ctx, "create_sample", func(tx domain.Transaction) error {
		var innerErr error
		created, innerErr = tx.CreateSample(sample)
		return innerErr
	})
	if err == nil {
		s.recordAuditSuccess(ctx, "create_sample", created.ID, dur)
	}
	return created, res, err
}

// UpdateSample mutates a sample record.
func (s *Service) UpdateSample(ctx context.Context, id string, mutator func(*domain.Sample) error) (domain.Sample, domain.Result, error) {
	var updated domain.Sample
	res, dur, err := s.run(ctx, "update_sample", func(tx domain.Transaction) error {
		var innerErr error
		updated, innerErr = tx.UpdateSample(id, mutator)
		return innerErr
	})
	if err == nil {
		s.recordAuditSuccess(ctx, "update_sample", updated.ID, dur)
	}
	return updated, res, err
}

// DeleteSample removes a sample.
func (s *Service) DeleteSample(ctx context.Context, id string) (domain.Result, error) {
	res, dur, err := s.run(ctx, "delete_sample", func(tx domain.Transaction) error {
		return tx.DeleteSample(id)
	})
	if err == nil {
		s.recordAuditSuccess(ctx, "delete_sample", id, dur)
	}
	return res, err
}

// CreatePermit persists a permit.
func (s *Service) CreatePermit(ctx context.Context, permit domain.Permit) (domain.Permit, domain.Result, error) {
	var created domain.Permit
	res, dur, err := s.run(ctx, "create_permit", func(tx domain.Transaction) error {
		var innerErr error
		created, innerErr = tx.CreatePermit(permit)
		return innerErr
	})
	if err == nil {
		s.recordAuditSuccess(ctx, "create_permit", created.ID, dur)
	}
	return created, res, err
}

// UpdatePermit mutates a permit record.
func (s *Service) UpdatePermit(ctx context.Context, id string, mutator func(*domain.Permit) error) (domain.Permit, domain.Result, error) {
	var updated domain.Permit
	res, dur, err := s.run(ctx, "update_permit", func(tx domain.Transaction) error {
		var innerErr error
		updated, innerErr = tx.UpdatePermit(id, mutator)
		return innerErr
	})
	if err == nil {
		s.recordAuditSuccess(ctx, "update_permit", updated.ID, dur)
	}
	return updated, res, err
}

// DeletePermit removes a permit.
func (s *Service) DeletePermit(ctx context.Context, id string) (domain.Result, error) {
	res, dur, err := s.run(ctx, "delete_permit", func(tx domain.Transaction) error {
		return tx.DeletePermit(id)
	})
	if err == nil {
		s.recordAuditSuccess(ctx, "delete_permit", id, dur)
	}
	return res, err
}

// CreateSupplyItem persists a supply item record.
func (s *Service) CreateSupplyItem(ctx context.Context, item domain.SupplyItem) (domain.SupplyItem, domain.Result, error) {
	var created domain.SupplyItem
	res, dur, err := s.run(ctx, "create_supply_item", func(tx domain.Transaction) error {
		var innerErr error
		created, innerErr = tx.CreateSupplyItem(item)
		return innerErr
	})
	if err == nil {
		s.recordAuditSuccess(ctx, "create_supply_item", created.ID, dur)
	}
	return created, res, err
}

// UpdateSupplyItem mutates a supply item.
func (s *Service) UpdateSupplyItem(ctx context.Context, id string, mutator func(*domain.SupplyItem) error) (domain.SupplyItem, domain.Result, error) {
	var updated domain.SupplyItem
	res, dur, err := s.run(ctx, "update_supply_item", func(tx domain.Transaction) error {
		var innerErr error
		updated, innerErr = tx.UpdateSupplyItem(id, mutator)
		return innerErr
	})
	if err == nil {
		s.recordAuditSuccess(ctx, "update_supply_item", updated.ID, dur)
	}
	return updated, res, err
}

// DeleteSupplyItem removes a supply item.
func (s *Service) DeleteSupplyItem(ctx context.Context, id string) (domain.Result, error) {
	res, dur, err := s.run(ctx, "delete_supply_item", func(tx domain.Transaction) error {
		return tx.DeleteSupplyItem(id)
	})
	if err == nil {
		s.recordAuditSuccess(ctx, "delete_supply_item", id, dur)
	}
	return res, err
}

// ErrNotFound is returned when reference validation fails within transactional helpers.
type ErrNotFound struct {
	Entity domain.EntityType
	ID     string
}

func (e ErrNotFound) Error() string {
	return fmt.Sprintf("%s %s not found", e.Entity, e.ID)
}

// InstallPlugin registers a plugin, wiring its rules into the active engine.
func (s *Service) InstallPlugin(plugin pluginapi.Plugin) (PluginMetadata, error) {
	if plugin == nil {
		return PluginMetadata{}, fmt.Errorf("plugin cannot be nil")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.plugins[plugin.Name()]; ok {
		return PluginMetadata{}, fmt.Errorf("plugin %s already registered", plugin.Name())
	}

	pluginMajor, pluginDeclared := entityModelMajorFromPlugin(plugin)
	if pluginDeclared {
		if err := requireEntityModelCompatibility(pluginMajor, fmt.Sprintf("plugin %s", plugin.Name())); err != nil {
			return PluginMetadata{}, err
		}
	}

	registry := NewPluginRegistry()
	if err := plugin.Register(registry); err != nil {
		return PluginMetadata{}, err
	}

	for _, rule := range registry.Rules() {
		if s.engine != nil {
			s.engine.Register(rule)
		}
	}

	meta := PluginMetadata{
		Name:    plugin.Name(),
		Version: plugin.Version(),
		Schemas: registry.Schemas(),
	}

	env := DatasetEnvironment{Store: s.store, Now: s.now}

	for _, dataset := range registry.DatasetTemplates() {
		dataset.Plugin = plugin.Name()
		templateMajor, templateDeclared := entityModelMajorFromTemplate(dataset.Template)
		if err := ensureTemplateCompatibility(templateMajor, templateDeclared, pluginMajor, pluginDeclared, dataset.slug()); err != nil {
			return PluginMetadata{}, err
		}
		if templateDeclared {
			if err := requireEntityModelCompatibility(templateMajor, fmt.Sprintf("dataset template %s", dataset.slug())); err != nil {
				return PluginMetadata{}, err
			}
		}
		if err := dataset.bind(env); err != nil {
			return PluginMetadata{}, fmt.Errorf("bind dataset %s: %w", dataset.Key, err)
		}
		slug := dataset.slug()
		if _, exists := s.datasets[slug]; exists {
			return PluginMetadata{}, fmt.Errorf("dataset template %s already installed", slug)
		}
		s.datasets[slug] = dataset
		meta.Datasets = append(meta.Datasets, dataset.Descriptor())
	}

	if len(meta.Datasets) > 0 {
		datasetapi.SortTemplateDescriptors(meta.Datasets)
	}

	s.plugins[plugin.Name()] = meta
	s.logger.Info("plugin installed", "plugin", plugin.Name(), "version", plugin.Version(), "datasets", len(meta.Datasets))
	return meta, nil
}

// RegisteredPlugins returns metadata describing installed plugins.
func (s *Service) RegisteredPlugins() []PluginMetadata {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]PluginMetadata, 0, len(s.plugins))
	for _, meta := range s.plugins {
		copyMeta := meta
		if len(meta.Datasets) > 0 {
			copyMeta.Datasets = append([]datasetapi.TemplateDescriptor(nil), meta.Datasets...)
		}
		if len(meta.Schemas) > 0 {
			schemaCopy := make(map[string]map[string]any, len(meta.Schemas))
			for k, v := range meta.Schemas {
				inner := make(map[string]any, len(v))
				for key, val := range v {
					inner[key] = val
				}
				schemaCopy[k] = inner
			}
			copyMeta.Schemas = schemaCopy
		}
		out = append(out, copyMeta)
	}
	return out
}

// DatasetTemplates returns all installed dataset template descriptors.
func (s *Service) DatasetTemplates() []datasetapi.TemplateDescriptor {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]datasetapi.TemplateDescriptor, 0, len(s.datasets))
	for _, template := range s.datasets {
		out = append(out, template.Descriptor())
	}
	datasetapi.SortTemplateDescriptors(out)
	return out
}

// ResolveDatasetTemplate fetches a dataset template by slug.
func (s *Service) ResolveDatasetTemplate(slug string) (datasetapi.TemplateRuntime, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	template, ok := s.datasets[slug]
	if !ok {
		return nil, false
	}
	return newDatasetTemplateRuntime(template), true
}

func (s *Service) recordAuditSuccess(ctx context.Context, op, entityID string, duration time.Duration) {
	meta := lookupOperationMeta(op)
	if meta.entity == "" {
		return
	}
	timestamp := time.Now().UTC()
	if s.now != nil {
		timestamp = s.now()
	}
	entry := AuditEntry{
		Operation: op,
		Entity:    meta.entity,
		Action:    meta.action,
		EntityID:  entityID,
		Status:    AuditStatusSuccess,
		Duration:  duration,
		Timestamp: timestamp,
	}
	s.audit.Record(ctx, entry)
}

func (s *Service) recordAuditFailure(ctx context.Context, op string, meta operationMeta, err error, duration time.Duration) {
	timestamp := time.Now().UTC()
	if s.now != nil {
		timestamp = s.now()
	}
	entry := AuditEntry{
		Operation: op,
		Entity:    meta.entity,
		Action:    meta.action,
		Status:    AuditStatusError,
		Duration:  duration,
		Timestamp: timestamp,
	}
	if err != nil {
		entry.Error = err.Error()
	}
	s.audit.Record(ctx, entry)
}

type operationMeta struct {
	entity domain.EntityType
	action domain.Action
}

func lookupOperationMeta(op string) operationMeta {
	if meta, ok := operationMetadata[op]; ok {
		return meta
	}
	return operationMeta{}
}

var operationMetadata = map[string]operationMeta{
	"create_project":           {entity: domain.EntityProject, action: domain.ActionCreate},
	"update_project":           {entity: domain.EntityProject, action: domain.ActionUpdate},
	"delete_project":           {entity: domain.EntityProject, action: domain.ActionDelete},
	"create_protocol":          {entity: domain.EntityProtocol, action: domain.ActionCreate},
	"update_protocol":          {entity: domain.EntityProtocol, action: domain.ActionUpdate},
	"delete_protocol":          {entity: domain.EntityProtocol, action: domain.ActionDelete},
	"create_facility":          {entity: domain.EntityFacility, action: domain.ActionCreate},
	"update_facility":          {entity: domain.EntityFacility, action: domain.ActionUpdate},
	"delete_facility":          {entity: domain.EntityFacility, action: domain.ActionDelete},
	"create_housing_unit":      {entity: domain.EntityHousingUnit, action: domain.ActionCreate},
	"update_housing_unit":      {entity: domain.EntityHousingUnit, action: domain.ActionUpdate},
	"delete_housing_unit":      {entity: domain.EntityHousingUnit, action: domain.ActionDelete},
	"create_cohort":            {entity: domain.EntityCohort, action: domain.ActionCreate},
	"create_organism":          {entity: domain.EntityOrganism, action: domain.ActionCreate},
	"update_organism":          {entity: domain.EntityOrganism, action: domain.ActionUpdate},
	"delete_organism":          {entity: domain.EntityOrganism, action: domain.ActionDelete},
	"assign_organism_housing":  {entity: domain.EntityOrganism, action: domain.ActionUpdate},
	"assign_organism_protocol": {entity: domain.EntityOrganism, action: domain.ActionUpdate},
	"create_breeding_unit":     {entity: domain.EntityBreeding, action: domain.ActionCreate},
	"create_procedure":         {entity: domain.EntityProcedure, action: domain.ActionCreate},
	"update_procedure":         {entity: domain.EntityProcedure, action: domain.ActionUpdate},
	"delete_procedure":         {entity: domain.EntityProcedure, action: domain.ActionDelete},
	"create_treatment":         {entity: domain.EntityTreatment, action: domain.ActionCreate},
	"update_treatment":         {entity: domain.EntityTreatment, action: domain.ActionUpdate},
	"delete_treatment":         {entity: domain.EntityTreatment, action: domain.ActionDelete},
	"create_observation":       {entity: domain.EntityObservation, action: domain.ActionCreate},
	"update_observation":       {entity: domain.EntityObservation, action: domain.ActionUpdate},
	"delete_observation":       {entity: domain.EntityObservation, action: domain.ActionDelete},
	"create_sample":            {entity: domain.EntitySample, action: domain.ActionCreate},
	"update_sample":            {entity: domain.EntitySample, action: domain.ActionUpdate},
	"delete_sample":            {entity: domain.EntitySample, action: domain.ActionDelete},
	"create_permit":            {entity: domain.EntityPermit, action: domain.ActionCreate},
	"update_permit":            {entity: domain.EntityPermit, action: domain.ActionUpdate},
	"delete_permit":            {entity: domain.EntityPermit, action: domain.ActionDelete},
	"create_supply_item":       {entity: domain.EntitySupplyItem, action: domain.ActionCreate},
	"update_supply_item":       {entity: domain.EntitySupplyItem, action: domain.ActionUpdate},
	"delete_supply_item":       {entity: domain.EntitySupplyItem, action: domain.ActionDelete},
}

func (s *Service) run(ctx context.Context, op string, fn func(domain.Transaction) error) (domain.Result, time.Duration, error) {
	meta := lookupOperationMeta(op)
	start := time.Now()
	ctx, span := s.tracer.Start(ctx, op)
	res, err := s.store.RunInTransaction(ctx, fn)
	duration := time.Since(start)
	success := err == nil

	s.metrics.Observe(ctx, op, success, duration)
	span.End(err)

	if err != nil {
		s.recordAuditFailure(ctx, op, meta, err, duration)
		s.logger.Error("service operation failed", "op", op, "error", err)
		return res, duration, err
	}
	s.logger.Debug("service operation succeeded", "op", op)
	return res, duration, nil
}

type rulesEngineProvider interface {
	RulesEngine() *domain.RulesEngine
}

type nowFuncProvider interface {
	NowFunc() func() time.Time
}

func extractRulesEngine(store domain.PersistentStore) *domain.RulesEngine {
	if provider, ok := store.(rulesEngineProvider); ok {
		return provider.RulesEngine()
	}
	return nil
}

func selectNowFunc(store domain.PersistentStore, clock Clock) func() time.Time {
	if provider, ok := store.(nowFuncProvider); ok {
		if fn := provider.NowFunc(); fn != nil {
			return func() time.Time { return fn().UTC() }
		}
	}
	if clock != nil {
		return func() time.Time { return clock.Now().UTC() }
	}
	return func() time.Time { return time.Now().UTC() }
}
