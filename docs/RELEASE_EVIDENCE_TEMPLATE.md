# Release Evidence

- Release version:
- Date:
- Service:
- Owner:

## Quality Gate

- [ ] `make quality-gate` PASS
- [ ] CI baseline (`make test`, `make vet`, `make lint`) PASS
- [ ] `docs/CHANGE_CHECKLIST.md` reviewed
- Notes:

## Smoke Gate (staging)

- [ ] `/health` PASS
- [ ] `/ready` PASS
- [ ] `/version` PASS
- [ ] `/version` matches injected `version`, `commit`, and `build_date`
- Base URL:
- Notes:

## Performance Gate (staging)

- [ ] Steady PASS
- [ ] Spike PASS
- [ ] Soak PASS
- Target path:
- Thresholds:
- Evidence links/screenshots:

## Failure Drill Gate

- [ ] DB failure/recovery PASS
- [ ] Kafka failure/recovery PASS
- [ ] OTLP failure/recovery PASS (if applicable)
- Notes:

## Security Gate

- [ ] TLS enabled in production
- [ ] `OTEL_EXPORTER_OTLP_INSECURE=false`
- [ ] Secrets from secret manager (not env file in repo)
- [ ] No critical security findings
- Notes:

## Observability Gate

- [ ] Dashboard updated
- [ ] Alerts tested
- [ ] Runbook linked
- Links:

## Rollback Readiness

- [ ] Rollback command tested
- [ ] Previous version artifact available
- Notes:

## Final Decision

- [ ] APPROVED FOR PRODUCTION
- Release version:
- Release commit:
- Build date:
- Version endpoint evidence:
- Approver:
- Date:
