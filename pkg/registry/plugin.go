package registry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
)

// pluginAttributeSpec is the JSON-serialisable form of an attribute spec used
// in plugin component definition files.
type pluginAttributeSpec struct {
	Type     string   `json:"type"`
	Required bool     `json:"required,omitempty"`
	Enum     []string `json:"enum,omitempty"`
	DocType  string   `json:"doc_type,omitempty"`
}

// pluginComponentSpec is the JSON schema for a single component entry inside a
// plugin definition file.
type pluginComponentSpec struct {
	Name        string                         `json:"name"`
	HasChildren bool                           `json:"has_children,omitempty"`
	Attributes  map[string]pluginAttributeSpec `json:"attributes"`
}

// pluginFile is the top-level structure of a .json plugin definition file.
type pluginFile struct {
	Components []pluginComponentSpec `json:"components"`
}

var (
	// pluginCollisions records component names from plugins that collide with
	// core component names.
	pluginCollisions []string
	// coreNames is the frozen set of component names defined before any plugin
	// is loaded.  It is initialised once from the initial componentSpecs map.
	coreNames map[string]struct{}
)

func init() {
	// Snapshot core component names before any plugins are loaded.
	coreNames = make(map[string]struct{}, len(componentSpecs))
	for name := range componentSpecs {
		coreNames[name] = struct{}{}
	}
	// Eagerly scan the ./components directory relative to the current working
	// directory.  A missing directory is silently ignored so that unit tests and
	// library users are unaffected.
	pluginCollisions = loadPluginsFromDir("components")
}

// LoadPluginsFromDir merges component definitions from all .json files found
// directly inside dir into the global componentSpecs map and returns the names
// of any core/plugin naming collisions.  It is intended for testing; normal
// callers rely on the init() auto-scan.
func LoadPluginsFromDir(dir string) []string {
	return loadPluginsFromDir(dir)
}

func loadPluginsFromDir(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		// Missing directory is not an error condition.
		return nil
	}

	var collisions []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		cols := loadPluginFile(path)
		collisions = append(collisions, cols...)
	}

	sort.Strings(collisions)
	return collisions
}

func loadPluginFile(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var pf pluginFile
	if err := json.Unmarshal(data, &pf); err != nil {
		return nil
	}

	var collisions []string
	for _, pc := range pf.Components {
		if pc.Name == "" {
			continue
		}
		if _, isCore := coreNames[pc.Name]; isCore {
			collisions = append(collisions, pc.Name)
			// Still load the plugin so the registry reflects the latest definition,
			// but flag the collision for the doctor check.
		}
		spec := convertPluginComponent(pc)
		componentSpecs[pc.Name] = spec
	}
	return collisions
}

func convertPluginComponent(pc pluginComponentSpec) ComponentSpec {
	attrs := make(map[string]AttributeSpec, len(pc.Attributes))
	for name, pa := range pc.Attributes {
		vt := parsePluginValueType(pa.Type)
		as := AttributeSpec{
			Type:     vt,
			Required: pa.Required,
			DocType:  pa.DocType,
		}
		if len(pa.Enum) > 0 {
			as.Enum = enumSet(pa.Enum...)
		}
		attrs[name] = as
	}

	return ComponentSpec{
		Name:        pc.Name,
		HasChildren: pc.HasChildren,
		Attributes:  attrs,
	}
}

func parsePluginValueType(raw string) ValueType {
	switch ValueType(raw) {
	case StringType, BoolType, IntType, NumberType,
		UnitType, UnitArrayType, StringArrayType,
		NumberArrayOrReferenceType, ActionType:
		return ValueType(raw)
	default:
		return StringType
	}
}

// RegistryCollisions returns the list of component names that were declared by
// a plugin but collide with a core component name.
func RegistryCollisions() []string {
	return append([]string(nil), pluginCollisions...)
}

// PluginComponentNames returns the sorted names of components contributed by
// plugins (i.e. names not in the core registry).
func PluginComponentNames() []string {
	var names []string
	for name := range componentSpecs {
		if _, isCore := coreNames[name]; !isCore {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

// PluginExampleContent is the content of the example plugin file written by
// corec init.
const PluginExampleContent = `{
  "components": [
    {
      "name": "Rating",
      "has_children": false,
      "attributes": {
        "id":        { "type": "string",  "required": true },
        "value":     { "type": "int" },
        "max":       { "type": "int" },
        "on_change": { "type": "action" },
        "hidden":    { "type": "bool" },
        "style":     { "type": "string" }
      }
    }
  ]
}
`

// PluginExampleName is the file name for the generated plugin example.
const PluginExampleName = "plugin_example.json"
