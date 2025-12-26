# Benchmark baselines

`baseline.withmeta.results` is the single comparison baseline for the Sweet
suite used in PR benchmarking.

Update workflow:
1. `CONF=baseline COUNT=10 scripts/benchmarks/run_sweet.sh`
2. `CONF=baseline scripts/benchmarks/aggregate_results.sh`
3. `CONF=baseline scripts/benchmarks/withmeta.sh`
4. `cp benchmarks/artifacts/baseline.withmeta.results internal/ci/benchmarks/baseline.withmeta.results`

Keep the baseline file updated intentionally; it is used as the reference for
benchstat comparisons in CI.
