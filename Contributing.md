# Contributing

We build a MPA with server side rendering. We have select JS but try to avoid it if not necissary.

## Tools
we use SQLC to generate go code for sql queries.
- the migrations are stored in /static/migrations/*.sql
- the queries are stored in /sqlc/queries/*.sql

we use Templ to write templates for server side rendering
- templ files are written in templ syntax and generate go files.

we use TailwindCSS for styling
we use daisyUI as the base component lib.

All the generation can be done with make, this command will generate sqlc, templ and tailwind.
```
make generate
```

## Organisation

We organise code according to "group by feature" rather than layer.

### Feature Implementation Pattern

When implementing a new feature (like special_dates), follow this pattern:

#### 1. Database Layer
- Add migration in `/static/migrations/*.sql`
- Add queries in `/sqlc/queries/*.sql` (usually in account.sql for now)
- Run `make generate` to generate Go code

#### 2. Core Business Logic
Create `internal/core/{feature_name}.go` with:
- Page handlers (e.g., `{Feature}Page`, `{Feature}NewPage`, `{Feature}EditPage`)
- CRUD handlers (e.g., `Handler{Feature}Upsert`, `Handler{Feature}Delete`)
- Domain types (e.g., `{Feature}`, `{Feature}Input`)
- Business logic functions (e.g., `Get{Feature}`, `Upsert{Feature}`, `Delete{Feature}`, `List{Feature}s`)

#### 3. Views/Templates
Create `internal/core/{feature_name}_view.templ` with:
- `Page{Feature}s(child)` - Layout wrapper
- `{Feature}sView(items)` - List view with table
- `PageEdit{Feature}(child)` - Edit layout wrapper  
- `{Feature}EditView(item)` - Edit form view
- `Delete{Feature}Button(id)` - Delete button component
- `{Feature}Form(item)` - Form component

#### 4. Navigation Integration
- Add navigation item in `internal/core/view_main.templ`
- Add icon in `internal/core/view_icons.templ` if needed
- Add routes in `internal/core/handler.go`

#### 5. Code Generation
- Run `make generate` to generate templ files
- Test with `go build ./...`
- Test functionality with local server

#### 6. Testing Checklist
- [ ] List view shows empty state correctly
- [ ] Create functionality works
- [ ] Read functionality works
- [ ] Update functionality works
- [ ] Delete functionality works
- [ ] Navigation link appears in sidebar
- [ ] All CRUD operations tested via HTTP requests
