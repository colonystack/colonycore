// Package plugins hosts plugin implementation subpackages. It intentionally
// contains no production runtime code itself; this file exists to satisfy
// tooling (import-boss, go vet) for the architectural guard tests that live
// alongside it.
//
// A NOTE ON testhelper:
//   The subpackage plugins/testhelper is a deliberate escape hatch used only
//   in tests to construct facade fixtures from internal domain entities. It is
//   excluded from the architecture test that forbids importing colonycore/pkg/domain
//   so that real plugin packages (e.g. plugins/frog) remain fully decoupled
//   from internal domain shapes. Do not import testhelper in production plugin
//   code; its presence is solely to aid unit tests and may change without
//   stability guarantees.
package plugins
