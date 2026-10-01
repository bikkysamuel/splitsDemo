# Both codebases follow a clean-architecture dependency rule, enforced by tooling

**iOS**: four layers as separate Swift packages, `Presentation → Domain ← Data → Infrastructure`, wired together only in the `SplitsApp` composition root. Domain is pure Swift with no SwiftUI, networking or Firebase. Views and ViewModels never see generated API types or Firebase. SPM package dependencies make the compiler enforce the rule.

**Go**: each domain package (`groups`, `expenses`, …) holds its entities, a service (its use cases) and the repository interface it needs. `store` implements those interfaces and `httpapi` calls the services. Domain packages never import `store`, `httpapi`, pgx or `net/http`, which golangci-lint's depguard enforces. Domain code is kept together per feature, not split into project-wide `entity/` and `usecase/` trees.

Chosen by the project owner for clear organisation and testability. The trade-off: because the server owns all business logic (ADR-0006) and the app stores no data (ADR-0005), the iOS Domain layer is thin, so the rule is applied for consistent boundaries rather than to protect heavy client logic.

## Consequences

- **iOS use cases exist only where they add something**: they combine more than one repository call, or are triggered from more than one entry point (for example, Approve from both a screen and a lock-screen notification action). Otherwise ViewModels call repository protocols directly. `/code-review` checks this rule.
- **Generated API types never leave `Data`**: mappers turn them into Domain entities, each mapper has tests, and invalid server data becomes a typed error there rather than a crash in a view.
