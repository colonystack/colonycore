package core

import (
	"context"
	"fmt"
	"sort"
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

// ServiceOption configures optional dependencies for the Service constructor.
type ServiceOption func(*serviceOptions)

type serviceOptions struct {
	clock  Clock
	logger Logger
}

// WithClock overrides the default clock used by the service.
func WithClock(clock Clock) ServiceOption {
	return func(opts *serviceOptions) {
		if clock != nil {
			opts.clock = clock
		}
	}
}

// WithLogger injects a logger used by the service.
func WithLogger(logger Logger) ServiceOption {
	return func(opts *serviceOptions) {
		if logger != nil {
			opts.logger = logger
		}
	}
}

func defaultServiceOptions() serviceOptions {
	return serviceOptions{
		clock:  ClockFunc(func() time.Time { return time.Now().UTC() }),
		logger: noopLogger{},
	}
}

// Service orchestrates transactional operations, plugin registration, and dataset binding.
type Service struct {
	store    PersistentStore
	engine   *RulesEngine
	clock    Clock
	now      func() time.Time
	logger   Logger
	plugins  map[string]PluginMetadata
	datasets map[string]DatasetTemplate
	mu       sync.RWMutex
}

// NewService constructs a service backed by the supplied store.
func NewService(store PersistentStore, opts ...ServiceOption) *Service {
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
		plugins:  make(map[string]PluginMetadata),
		datasets: make(map[string]DatasetTemplate),
	}
	svc.engine = extractRulesEngine(store)
	svc.now = selectNowFunc(store, svc.clock)
	return svc
}

// NewInMemoryService creates a service and in-memory store with the given rules engine.
func NewInMemoryService(engine *RulesEngine, opts ...ServiceOption) *Service {
	store := NewMemoryStore(engine)
	return NewService(store, opts...)
}

// Store returns the underlying storage implementation.
func (s *Service) Store() PersistentStore {
	return s.store
}

// CreateProject persists a new project.
func (s *Service) CreateProject(ctx context.Context, project Project) (Project, Result, error) {
	var created Project
	res, err := s.run(ctx, "create_project", func(tx Transaction) error {
		var innerErr error
		created, innerErr = tx.CreateProject(project)
		return innerErr
	})
	return created, res, err
}

// CreateProtocol persists a new protocol.
func (s *Service) CreateProtocol(ctx context.Context, protocol Protocol) (Protocol, Result, error) {
	var created Protocol
	res, err := s.run(ctx, "create_protocol", func(tx Transaction) error {
		var innerErr error
		created, innerErr = tx.CreateProtocol(protocol)
		return innerErr
	})
	return created, res, err
}

// CreateHousingUnit persists housing metadata.
func (s *Service) CreateHousingUnit(ctx context.Context, housing HousingUnit) (HousingUnit, Result, error) {
	var created HousingUnit
	res, err := s.run(ctx, "create_housing_unit", func(tx Transaction) error {
		var innerErr error
		created, innerErr = tx.CreateHousingUnit(housing)
		return innerErr
	})
	return created, res, err
}

// CreateCohort persists a new cohort.
func (s *Service) CreateCohort(ctx context.Context, cohort Cohort) (Cohort, Result, error) {
	var created Cohort
	res, err := s.run(ctx, "create_cohort", func(tx Transaction) error {
		var innerErr error
		created, innerErr = tx.CreateCohort(cohort)
		return innerErr
	})
	return created, res, err
}

// CreateOrganism persists a new organism.
func (s *Service) CreateOrganism(ctx context.Context, organism Organism) (Organism, Result, error) {
	var created Organism
	res, err := s.run(ctx, "create_organism", func(tx Transaction) error {
		var innerErr error
		created, innerErr = tx.CreateOrganism(organism)
		return innerErr
	})
	return created, res, err
}

// UpdateOrganism mutates an organism using the provided mutator.
func (s *Service) UpdateOrganism(ctx context.Context, id string, mutator func(*Organism) error) (Organism, Result, error) {
	var updated Organism
	res, err := s.run(ctx, "update_organism", func(tx Transaction) error {
		var innerErr error
		updated, innerErr = tx.UpdateOrganism(id, mutator)
		return innerErr
	})
	return updated, res, err
}

// DeleteOrganism removes an organism record.
func (s *Service) DeleteOrganism(ctx context.Context, id string) (Result, error) {
	return s.run(ctx, "delete_organism", func(tx Transaction) error {
		return tx.DeleteOrganism(id)
	})
}

// AssignOrganismHousing updates an organism's housing reference within a transaction that validates dependencies.
func (s *Service) AssignOrganismHousing(ctx context.Context, organismID, housingID string) (Organism, Result, error) {
	var updated Organism
	res, err := s.run(ctx, "assign_organism_housing", func(tx Transaction) error {
		if _, ok := tx.FindHousingUnit(housingID); !ok {
			return ErrNotFound{Entity: EntityHousingUnit, ID: housingID}
		}
		var innerErr error
		updated, innerErr = tx.UpdateOrganism(organismID, func(o *Organism) error {
			o.HousingID = &housingID
			return nil
		})
		return innerErr
	})
	return updated, res, err
}

// AssignOrganismProtocol links an organism to a protocol within the same transactional scope.
func (s *Service) AssignOrganismProtocol(ctx context.Context, organismID, protocolID string) (Organism, Result, error) {
	var updated Organism
	res, err := s.run(ctx, "assign_organism_protocol", func(tx Transaction) error {
		if _, ok := tx.FindProtocol(protocolID); !ok {
			return ErrNotFound{Entity: EntityProtocol, ID: protocolID}
		}
		var innerErr error
		updated, innerErr = tx.UpdateOrganism(organismID, func(o *Organism) error {
			o.ProtocolID = &protocolID
			return nil
		})
		return innerErr
	})
	return updated, res, err
}

// CreateBreedingUnit persists a breeding configuration.
func (s *Service) CreateBreedingUnit(ctx context.Context, unit BreedingUnit) (BreedingUnit, Result, error) {
	var created BreedingUnit
	res, err := s.run(ctx, "create_breeding_unit", func(tx Transaction) error {
		var innerErr error
		created, innerErr = tx.CreateBreedingUnit(unit)
		return innerErr
	})
	return created, res, err
}

// CreateProcedure persists a procedure record.
func (s *Service) CreateProcedure(ctx context.Context, procedure Procedure) (Procedure, Result, error) {
	var created Procedure
	res, err := s.run(ctx, "create_procedure", func(tx Transaction) error {
		var innerErr error
		created, innerErr = tx.CreateProcedure(procedure)
		return innerErr
	})
	return created, res, err
}

// UpdateProcedure mutates a procedure.
func (s *Service) UpdateProcedure(ctx context.Context, id string, mutator func(*Procedure) error) (Procedure, Result, error) {
	var updated Procedure
	res, err := s.run(ctx, "update_procedure", func(tx Transaction) error {
		var innerErr error
		updated, innerErr = tx.UpdateProcedure(id, mutator)
		return innerErr
	})
	return updated, res, err
}

// DeleteProcedure removes a procedure record.
func (s *Service) DeleteProcedure(ctx context.Context, id string) (Result, error) {
	return s.run(ctx, "delete_procedure", func(tx Transaction) error {
		return tx.DeleteProcedure(id)
	})
}

// ErrNotFound is returned when reference validation fails within transactional helpers.
type ErrNotFound struct {
	Entity EntityType
	ID     string
}

func (e ErrNotFound) Error() string {
	return fmt.Sprintf("%s %s not found", e.Entity, e.ID)
}

// InstallPlugin registers a plugin, wiring its rules into the active engine.
func (s *Service) InstallPlugin(plugin Plugin) (PluginMetadata, error) {
	if plugin == nil {
		return PluginMetadata{}, fmt.Errorf("plugin cannot be nil")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.plugins[plugin.Name()]; ok {
		return PluginMetadata{}, fmt.Errorf("plugin %s already registered", plugin.Name())
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
	if env.Now == nil {
		env.Now = func() time.Time { return time.Now().UTC() }
	}

	for _, dataset := range registry.DatasetTemplates() {
		dataset.Plugin = plugin.Name()
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
		sort.Sort(DatasetTemplateCollection(meta.Datasets))
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
			copyMeta.Datasets = append([]DatasetTemplateDescriptor(nil), meta.Datasets...)
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
func (s *Service) DatasetTemplates() []DatasetTemplateDescriptor {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]DatasetTemplateDescriptor, 0, len(s.datasets))
	for _, template := range s.datasets {
		out = append(out, template.Descriptor())
	}
	sort.Sort(DatasetTemplateCollection(out))
	return out
}

// ResolveDatasetTemplate fetches a dataset template by slug.
func (s *Service) ResolveDatasetTemplate(slug string) (DatasetTemplate, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	template, ok := s.datasets[slug]
	return template, ok
}

func (s *Service) run(ctx context.Context, op string, fn func(Transaction) error) (Result, error) {
	res, err := s.store.RunInTransaction(ctx, fn)
	if err != nil {
		s.logger.Error("service operation failed", "op", op, "error", err)
		return res, err
	}
	s.logger.Debug("service operation succeeded", "op", op)
	return res, nil
}

type rulesEngineProvider interface {
	RulesEngine() *RulesEngine
}

type nowFuncProvider interface {
	NowFunc() func() time.Time
}

func extractRulesEngine(store PersistentStore) *RulesEngine {
	if provider, ok := store.(rulesEngineProvider); ok {
		return provider.RulesEngine()
	}
	return nil
}

func selectNowFunc(store PersistentStore, clock Clock) func() time.Time {
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
