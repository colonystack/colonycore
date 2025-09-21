package core

import (
	"context"
	"fmt"
)

// Service exposes higher-level transactional CRUD operations for the core schema.
type Service struct {
	store   *MemoryStore
	plugins map[string]PluginMetadata
}

// NewService constructs a service backed by the supplied store.
func NewService(store *MemoryStore) *Service {
	return &Service{
		store:   store,
		plugins: make(map[string]PluginMetadata),
	}
}

// NewInMemoryService creates a service and in-memory store with the given rules engine.
func NewInMemoryService(engine *RulesEngine) *Service {
	store := NewMemoryStore(engine)
	return &Service{
		store:   store,
		plugins: make(map[string]PluginMetadata),
	}
}

// Store returns the underlying storage implementation.
func (s *Service) Store() *MemoryStore {
	return s.store
}

// CreateProject persists a new project.
func (s *Service) CreateProject(ctx context.Context, project Project) (Project, Result, error) {
	var created Project
	res, err := s.store.RunInTransaction(ctx, func(tx *Transaction) error {
		var err error
		created, err = tx.CreateProject(project)
		return err
	})
	return created, res, err
}

// CreateProtocol persists a new protocol.
func (s *Service) CreateProtocol(ctx context.Context, protocol Protocol) (Protocol, Result, error) {
	var created Protocol
	res, err := s.store.RunInTransaction(ctx, func(tx *Transaction) error {
		var err error
		created, err = tx.CreateProtocol(protocol)
		return err
	})
	return created, res, err
}

// CreateHousingUnit persists housing metadata.
func (s *Service) CreateHousingUnit(ctx context.Context, housing HousingUnit) (HousingUnit, Result, error) {
	var created HousingUnit
	res, err := s.store.RunInTransaction(ctx, func(tx *Transaction) error {
		var err error
		created, err = tx.CreateHousingUnit(housing)
		return err
	})
	return created, res, err
}

// CreateCohort persists a new cohort.
func (s *Service) CreateCohort(ctx context.Context, cohort Cohort) (Cohort, Result, error) {
	var created Cohort
	res, err := s.store.RunInTransaction(ctx, func(tx *Transaction) error {
		var err error
		created, err = tx.CreateCohort(cohort)
		return err
	})
	return created, res, err
}

// CreateOrganism persists a new organism.
func (s *Service) CreateOrganism(ctx context.Context, organism Organism) (Organism, Result, error) {
	var created Organism
	res, err := s.store.RunInTransaction(ctx, func(tx *Transaction) error {
		var err error
		created, err = tx.CreateOrganism(organism)
		return err
	})
	return created, res, err
}

// UpdateOrganism mutates an organism using the provided mutator.
func (s *Service) UpdateOrganism(ctx context.Context, id string, mutator func(*Organism) error) (Organism, Result, error) {
	var updated Organism
	res, err := s.store.RunInTransaction(ctx, func(tx *Transaction) error {
		var err error
		updated, err = tx.UpdateOrganism(id, mutator)
		return err
	})
	return updated, res, err
}

// DeleteOrganism removes an organism record.
func (s *Service) DeleteOrganism(ctx context.Context, id string) (Result, error) {
	res, err := s.store.RunInTransaction(ctx, func(tx *Transaction) error {
		return tx.DeleteOrganism(id)
	})
	return res, err
}

// AssignOrganismHousing updates an organism's housing reference within a transaction that validates dependencies.
func (s *Service) AssignOrganismHousing(ctx context.Context, organismID, housingID string) (Organism, Result, error) {
	var updated Organism
	res, err := s.store.RunInTransaction(ctx, func(tx *Transaction) error {
		if _, ok := tx.state.housing[housingID]; !ok {
			return ErrNotFound{Entity: EntityHousingUnit, ID: housingID}
		}
		var err error
		updated, err = tx.UpdateOrganism(organismID, func(o *Organism) error {
			o.HousingID = &housingID
			return nil
		})
		return err
	})
	return updated, res, err
}

// AssignOrganismProtocol links an organism to a protocol within the same transactional scope.
func (s *Service) AssignOrganismProtocol(ctx context.Context, organismID, protocolID string) (Organism, Result, error) {
	var updated Organism
	res, err := s.store.RunInTransaction(ctx, func(tx *Transaction) error {
		if _, ok := tx.state.protocols[protocolID]; !ok {
			return ErrNotFound{Entity: EntityProtocol, ID: protocolID}
		}
		var err error
		updated, err = tx.UpdateOrganism(organismID, func(o *Organism) error {
			o.ProtocolID = &protocolID
			return nil
		})
		return err
	})
	return updated, res, err
}

// CreateBreedingUnit persists a breeding configuration.
func (s *Service) CreateBreedingUnit(ctx context.Context, unit BreedingUnit) (BreedingUnit, Result, error) {
	var created BreedingUnit
	res, err := s.store.RunInTransaction(ctx, func(tx *Transaction) error {
		var err error
		created, err = tx.CreateBreedingUnit(unit)
		return err
	})
	return created, res, err
}

// CreateProcedure persists a procedure record.
func (s *Service) CreateProcedure(ctx context.Context, procedure Procedure) (Procedure, Result, error) {
	var created Procedure
	res, err := s.store.RunInTransaction(ctx, func(tx *Transaction) error {
		var err error
		created, err = tx.CreateProcedure(procedure)
		return err
	})
	return created, res, err
}

// UpdateProcedure mutates a procedure.
func (s *Service) UpdateProcedure(ctx context.Context, id string, mutator func(*Procedure) error) (Procedure, Result, error) {
	var updated Procedure
	res, err := s.store.RunInTransaction(ctx, func(tx *Transaction) error {
		var err error
		updated, err = tx.UpdateProcedure(id, mutator)
		return err
	})
	return updated, res, err
}

// DeleteProcedure removes a procedure record.
func (s *Service) DeleteProcedure(ctx context.Context, id string) (Result, error) {
	res, err := s.store.RunInTransaction(ctx, func(tx *Transaction) error {
		return tx.DeleteProcedure(id)
	})
	return res, err
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
	if s.plugins == nil {
		s.plugins = make(map[string]PluginMetadata)
	}
	if _, ok := s.plugins[plugin.Name()]; ok {
		return PluginMetadata{}, fmt.Errorf("plugin %s already registered", plugin.Name())
	}

	registry := NewPluginRegistry()
	if err := plugin.Register(registry); err != nil {
		return PluginMetadata{}, err
	}

	for _, rule := range registry.Rules() {
		s.store.engine.Register(rule)
	}

	meta := PluginMetadata{
		Name:    plugin.Name(),
		Version: plugin.Version(),
		Schemas: registry.Schemas(),
	}
	s.plugins[plugin.Name()] = meta
	return meta, nil
}

// RegisteredPlugins returns metadata describing installed plugins.
func (s *Service) RegisteredPlugins() []PluginMetadata {
	out := make([]PluginMetadata, 0, len(s.plugins))
	for _, meta := range s.plugins {
		out = append(out, meta)
	}
	return out
}
