# Stack Migrate Frontend

Migrate a frontend POC (React, Next.js, Vite, etc.) to use the stack's backend and frontend patterns. Analyzes the POC, identifies the right migration path, and produces a concrete plan.

## Context

Frontend POCs are often built with React/Vite/Next.js to iterate quickly — heavy client-side frameworks with lots of npm dependencies. When it's time to productionize, we want to migrate them onto our stack: Go backend (GoAppLib), tsappkit for frontend component lifecycle, and our auth/infra components. The goal is fewer dependencies, more control, and integration with the rest of our tooling.

These POCs range from simple dashboards to complex interactive apps (drag-and-drop editors, real-time collaboration, game-like UIs). The migration path depends on the app's complexity and interactivity needs.

## Migration Paths

### Path A: Full Stack Migration (GoAppLib + tsappkit + templates)
Best for: most POCs — even complex ones. This is the default path.
- Go backend with GoAppLib for routing, ViewContext, page mixins, auth integration
- Replace React component tree with tsappkit BaseComponent + LifecycleController
- Replace React state (Redux/Zustand/Context) with tsappkit EventBus + component-local state
- Replace React Router with Go HTTP handlers serving HTML + tsappkit component init
- Keep complex JS interactions as vanilla TS (animations, drag, canvas) — tsappkit's LCM coordinates them
- Use Templar for HTML templates with Go template inheritance
- Keep Tailwind (works with templates), keep design tokens
- Use devloop for live reload of both Go and TS

**Component mapping:**
| React Pattern | Stack Replacement |
|--------------|-------------------|
| React components | tsappkit BaseComponent (DOM-scoped, lifecycle-managed) |
| useState/useEffect | Component-local state + LCM phases (performLocalInit → activate) |
| useContext / Redux / Zustand | tsappkit EventBus for cross-component communication |
| React Router | Go HTTP handlers + GoAppLib ViewContext (server-side routing) |
| useRef + DOM manipulation | Direct DOM access (tsappkit components own their subtree) |
| Modals/Toasts | tsappkit Modal, ToastManager (singleton managers) |
| Theme switching | tsappkit ThemeManager (localStorage + system detection) |
| Keyboard shortcuts | tsappkit KeyboardShortcutManager (multi-key, state machine) |
| JSX templates | Go HTML templates via Templar (server-side) + jsx-dom for client-side element trees (real DOM, no virtual DOM) |
| IndexedDB / localStorage | Go backend + Redis/DB via stack (protoc-gen-dal for ORM) |
| fetch/axios API calls | HTMX for simple loads, fetch for complex interactions |
| CSS-in-JS / Tailwind | Keep Tailwind as-is (works with Go templates) |
| Vite HMR | devloop with live reload for Go + TS watch |

### Path B: Backend-Only Migration (Keep React, add Go backend)
Best for: when the React frontend is genuinely too complex to rewrite (rare — try Path A first).
- Keep the React app as-is but add a Go backend for persistence, auth, real-time
- Wire auth through OneAuth
- Use ServiceKit/MassRelay for WebSocket features
- Use devloop to run both React dev server and Go backend
- This is a stepping stone — plan to migrate frontend to tsappkit later

### Path C: WASM Bridge (protoc-gen-go-wasmjs)
Best for: when business logic must run in both server and browser (validation, computation, game rules).
- Define shared logic as protobuf services
- Generate Go WASM bindings + TypeScript clients
- Frontend (tsappkit or React) calls generated TS client
- Combine with Path A or B for the UI layer

## Steps

1. **Analyze the POC**: Read the POC's package.json, key source files, and directory structure. Identify:
   - Framework and build tooling (React/Vite/Next/etc.)
   - Component inventory — list all components with their role and interaction patterns
   - State management pattern and where state lives (client DB, server, URL, etc.)
   - API/data layer (REST, GraphQL, IndexedDB, localStorage, chrome APIs, etc.)
   - Auth approach
   - Real-time features (WebSocket, SSE, polling)
   - Third-party dependencies and what each one does
   - Non-web parts (Chrome extensions, Electron, mobile, etc.)

2. **Compatibility audit (Path A feasibility)**: Go through every component and dependency and classify it into one of these buckets. The goal is to prove Path A works by finding what *doesn't* fit — not to assume it works.

   **Bucket 1: Direct stack equivalent** — the stack already has this.
   List each item with: POC feature → stack replacement (component + specific capability).
   Examples: React Router → Go handlers, Zustand → EventBus, modals → tsappkit Modal, etc.

   **Bucket 2: Straightforward port** — no stack equivalent but easy to rewrite in vanilla TS + tsappkit + jsx-dom.
   List each item with: what it does, why it's straightforward (no deep framework coupling).
   Examples: custom drag handler, canvas drawing, animation loops, keyboard listeners, JSX-heavy render methods.
   Note: jsx-dom lets you keep JSX syntax for building element trees — it produces real DOM nodes, not React virtual DOM. So JSX-heavy components port directly (see jsx-dom section below for the pattern).

   **Bucket 3: React-ecosystem dependency** — the POC uses a React-specific library with no vanilla equivalent.
   List each item with: what library, what it does, whether a vanilla/non-React alternative exists.
   Examples: React Spring (→ Web Animations API or GSAP), React DnD (→ native drag API), Framer Motion, React Three Fiber (→ raw Three.js).
   For each, note whether the vanilla alternative is a reasonable migration or a significant rewrite.

   **Bucket 4: True blockers** — things that genuinely cannot work without React or a similar framework.
   These are rare. Examples might be: React Native (mobile), React Server Components (streaming SSR), or a third-party component library with no vanilla equivalent (e.g., a complex data grid with 50+ features).
   Be specific about *why* it's a blocker, not just that it's complex.

   **Present the audit as a table:**
   | POC Feature/Dep | Bucket | Stack/Vanilla Replacement | Migration Effort | Notes |
   |----------------|--------|--------------------------|-----------------|-------|

   If Bucket 4 is empty → recommend Path A with confidence.
   If Bucket 4 has items → present them to the user. Options:
   - Accept reduced functionality (drop the feature)
   - Build the missing capability (vanilla TS or new stack component)
   - Use Path B as interim for those specific areas
   - Keep a single React island for just that feature (micro-frontend)

3. **Check stack readiness**: Run `stack-brain lookup` for capabilities the POC needs. Identify:
   - Stack components that replace POC dependencies (feeds into Bucket 1)
   - Third-party deps that must be kept (no stack equivalent)
   - Stack gaps → log to ~/newstack/brain/STACK_GAPS.md

4. **Design the target architecture**: Based on the chosen path, produce:

   **Directory structure:**
   ```
   project/
   ├── cmd/server/main.go       # Go entry point
   ├── go.mod                   # Stack component dependencies
   ├── Stackfile.md             # Stack version tracking
   ├── views/                   # Go HTTP handlers (GoAppLib ViewContext)
   ├── templates/               # Go HTML templates (Templar)
   │   ├── layouts/             # Base layouts with template inheritance
   │   ├── pages/               # Page-level templates
   │   └── partials/            # Reusable fragments (HTMX targets)
   ├── ts/                      # TypeScript source (tsappkit components)
   │   ├── components/          # BaseComponent subclasses
   │   ├── events.ts            # EventBus event types
   │   └── index.ts             # Entry point, LifecycleController setup
   ├── static/                  # Built JS/CSS, images, fonts
   ├── devloop.yaml             # Dev orchestration config
   └── Makefile
   ```

   **Component migration map**: For each React component, specify:
   - Target: tsappkit BaseComponent, Go template partial, HTMX fragment, or vanilla TS
   - Which LCM phase handles its initialization
   - EventBus events it publishes/subscribes to
   - Any complex JS that stays as vanilla TS (with rationale)

   **Data layer migration**: Map client-side storage to server-side:
   - IndexedDB/localStorage → Go backend + database (protoc-gen-dal)
   - Zustand/Redux stores → EventBus + server state (fetched via HTMX or fetch)
   - URL state → Go router params + ViewContext

   **Non-web parts**: For things like Chrome extensions, keep them but rewire:
   - Extension talks to Go backend instead of IndexedDB
   - Or extension stays standalone, app imports via API endpoint instead of postMessage

5. **Produce the migration plan**: Phased, ordered:
   - **Phase 1: Scaffold** — Go module, Stackfile.md, devloop config, directory structure, tsappkit + Tailwind build pipeline
   - **Phase 2: Backend** — Go handlers that serve the same pages, data models, persistence layer
   - **Phase 3: Templates** — Convert React JSX to Go HTML templates + Templar inheritance. Get pages rendering server-side (even if not interactive yet)
   - **Phase 4: Components** — Migrate React components to tsappkit BaseComponents, one by one. Start with simple ones, build up. Wire EventBus for cross-component state
   - **Phase 5: Interactions** — Port complex JS (drag, animations, canvas) as vanilla TS, coordinated by tsappkit LCM
   - **Phase 6: Auth & Middleware** — OneAuth, rate limiting, CSRF via GoAppLib
   - **Phase 7: Cleanup** — Remove React/Vite/npm deps, verify feature parity, run /stack-audit

6. **Create Stackfile.md**: Draft a Stackfile.md listing all stack components that will be used.

7. **Confirm with user**: Present the full plan. Ask if they want to proceed with implementation or adjust.

## jsx-dom: JSX Without React

The key enabler for migrating JSX-heavy React code is **jsx-dom** — a library that makes JSX produce real DOM elements instead of React virtual DOM nodes. This means you keep the JSX syntax developers are familiar with but drop the entire React runtime (reconciler, fiber, hooks, etc.).

**Setup**: Add `/** @jsxImportSource jsx-dom */` at the top of .tsx files. Configure tsconfig to use jsx-dom. That's it.

**How it works**: `<div className="foo">` returns an actual `HTMLDivElement`. You `appendChild` it directly. Event handlers (`onClick`) attach real DOM listeners. No diffing, no re-rendering — just DOM construction.

**Pattern** — a plain TypeScript class using jsx-dom to build a detail page with Tailwind, event handlers, conditional rendering, and SVG icons. No React, no framework:
```typescript
/** @jsxImportSource jsx-dom */

class DetailPage {
  private container: HTMLElement;
  private store: DataStore;

  constructor() {
    this.store = new DataStore();
    this.container = document.getElementById('detail-root')!;
  }

  async init(): Promise<void> {
    const item = await this.store.getById(this.itemId);
    if (!item) { this.renderNotFound(); return; }
    this.renderDetail(item);
  }

  private renderDetail(item: Item): void {
    const view = (
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <h1 className="text-2xl font-bold">{item.title}</h1>
          <div className="flex gap-2">
            <a href={`/items/${item.id}/edit`}
               className="px-4 py-2 text-sm text-white bg-indigo-600 rounded-lg">
              Edit
            </a>
            <button className="px-4 py-2 text-sm text-red-600 bg-red-50 rounded-lg"
                    onClick={() => this.handleDelete(item)}>
              Delete
            </button>
          </div>
        </div>
        {item.preview
          ? <img src={item.preview} className="w-full rounded-xl" />
          : <div className="py-24 text-center text-gray-400">No preview</div>
        }
      </div>
    );
    this.container.appendChild(view);
  }

  private renderNotFound(): void {
    this.container.appendChild(
      <div className="text-center py-16">
        <h3>Not found</h3>
        <a href="/" className="mt-4 text-indigo-600">Back to list</a>
      </div>
    );
  }

  private async handleDelete(item: Item): Promise<void> {
    if (!confirm(`Delete "${item.title}"?`)) return;
    await this.store.delete(item.id);
    window.location.href = '/';
  }
}

document.addEventListener('DOMContentLoaded', () => new DetailPage().init());
```

**When to use jsx-dom vs Go templates**:
- Go templates (Templar): initial page structure, server-rendered content, SEO-relevant markup
- jsx-dom: dynamic client-side UI that gets built/rebuilt in response to user actions (modals, detail views, dynamic lists, interactive editors)

**When combined with tsappkit**: BaseComponent subclasses use jsx-dom in their `activate()` or render methods to build their DOM subtree. The LCM coordinates when components initialize; jsx-dom handles how they build their elements.

## tsappkit Migration Patterns

These patterns come up repeatedly when converting React to tsappkit:

**React useState → Component property + re-render helper**
```typescript
// React
const [count, setCount] = useState(0);

// tsappkit BaseComponent
class Counter extends BaseComponent {
  private count = 0;
  activate() {
    this.query('.increment')?.addEventListener('click', () => {
      this.count++;
      this.updateContent('.count-display', `${this.count}`);
    });
  }
}
```

**React useEffect → LCM phases**
```typescript
// React
useEffect(() => { fetchData(); return () => cleanup(); }, []);

// tsappkit
performLocalInit() { /* discover child elements */ }
activate() { this.fetchData(); }
deactivate() { this.cleanup(); }
```

**React Context/Redux → EventBus**
```typescript
// React Context
const theme = useContext(ThemeContext);

// tsappkit
this.eventBus.on('theme-changed', (theme) => this.applyTheme(theme));
```

**React component composition → LCM dependency injection**
```typescript
// tsappkit LifecycleController handles init order
// Phase 2 (setupDependencies) receives references to sibling components
setupDependencies(deps: Map<string, Component>) {
  this.timeline = deps.get('timeline') as TimelineComponent;
}
```

## Arguments
- $ARGUMENTS: Path to the POC directory. If omitted, uses current working directory. Can optionally include hints like "keep the extension as-is" or "prioritize the editor page first" to guide the plan.

## Notes
- This skill produces a migration plan — it does NOT auto-migrate. The user drives implementation.
- Complex interactive JS (drag, canvas, animations) does NOT need a framework. Vanilla TS + tsappkit LCM for coordination is the target — not reimplementing React.
- If the POC uses TypeScript, the migration preserves type safety — tsappkit is fully typed.
- Tailwind works identically with Go templates. No CSS migration needed.
- Chrome extensions, Electron shells, etc. are kept but rewired to talk to the Go backend.
- For POCs with no backend (e.g., IndexedDB-only apps), the biggest win is adding a real persistence layer via Go + protoc-gen-dal, not just swapping the UI framework.
