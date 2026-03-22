# {Component Name}

## Version
{version}

## Provides
- {capability-tag}: {one-line description}

## Module
{go module path or npm package name}

## Location
{path relative to ~ — e.g. ~/newstack/mylib/main}

## Stack Dependencies
- {component name} ({module path})
{or "None" if this is a leaf component}

## Integration

### Go Module
```go
// go.mod
require {module-path} {version}

// Local development
replace {module-path} => ~/newstack/{name}/{branch}
```

### Key Imports
```go
import "{module-path}/{subpackage}"
```

## Status
{Mature | Active | Stable | Planning | Basic}

## Conventions
- {key convention or pattern to follow when integrating}

## Migrations

{Leave empty until first version bump. Then add:}

### {old-version} → {new-version} ({date})
- **Breaking**: {what changed}
- **Migration**: See migrations/{old}_to_{new}.md
- **New**: {new capabilities added}
- **Deprecated**: {what's being phased out}
