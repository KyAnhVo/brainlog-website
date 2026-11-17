# Assumptions

These are decisions made while building `admin-panel-client` where the requirements in `requirements/`
were silent, ambiguous, or contradictory. Each file covers one area. Anything marked **ASSUMED API**
needs agreement with the server before it is real.

| File | Area |
|---|---|
| [api-endpoints.md](api-endpoints.md) | Endpoint paths and shapes that were not specified |
| [auth-session.md](auth-session.md) | Login, session detection, redirects, logout |
| [tracking-list.md](tracking-list.md) | `/tracking` table, filters, sort, pagination |
| [tracking-detail.md](tracking-detail.md) | `/tracking/$trackingNumber` page and status updates |
| [accounts.md](accounts.md) | `/accounts` create and delete |
| [ui-style.md](ui-style.md) | Look, structure, and code-style choices |
| [mocks.md](mocks.md) | How the MSW mock deviates from a real server |
