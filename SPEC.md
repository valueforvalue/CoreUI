# CoreUI Technical Specification v1.0

## 1. BNF Grammar
<program>    ::= [ <theme_block> ] <component>
<theme_block> ::= "Theme" "(" <attr_list> ")" "{" <theme_entries> "}"
<theme_entries> ::= <theme_entry> { [ "," ] <theme_entry> } | ""
<theme_entry> ::= "Color" "(" <attr_list> ")"
<component>  ::= <type> "(" <attr_list> ")" [ "{" <children> "}" ]
<type>       ::= [A-Z][a-zA-Z0-9]*
<attr_list>  ::= <attr> { "," <attr> } | ""
<attr>       ::= "action" "=" <action_val> | <key> "=" <value>
<key>        ::= [a-z][a-z0-9_]*
<value>      ::= <string> | <unit> | <number> | <boolean> | <array>
<unit>       ::= <number> ("px" | "%" | "*" | "auto")
<action_val> ::= <namespace> ":" <call> [ "(" <param_list> ")" ]

### Syntax Rules
1. **Bare Units in Arrays:** Units inside arrays such as `cols` or `rows` must not be quoted.
   * Correct: `cols=[1*, 200px]`
   * Incorrect: `cols=["1*", "200px"]`
2. **Double-Quote Totality:** All strings and action parameters must use double quotes (`"`) to maintain parity with Go and JSON standards.
   * Correct: `action="ui:navigate(target=\"settings\")"`
   * Note: CLI AI clients should handle escaping internal quotes automatically.

## 2. Component Registry
UI components support `id` (String - Mandatory), `hidden` (Bool), and `style` (String). `Theme` is a top-level metadata block, and `Color` entries define thematic token pairs via `key` and `value`.

| Component | Allowed Attributes |
| :--- | :--- |
| **Theme** | id |
| **Color** | key (String - Mandatory), value (String - Mandatory) |
| **View** | title, theme |
| **Stack** | dir ("h"|"v"), gap (Unit), align |
| **Grid** | cols (Array of Units), rows (Array of Units), gap (Unit) |
| **Box** | padding (Unit), border (Int), background |
| **Text** | value, size (Unit), weight |
| **Input** | type, label, bind |
| **Trigger** | label, action, variant |
| **DataTable**| source, selectable (Bool) |

## 3. The Action Protocol
Interactivity follows the format: `namespace:action(key=value)`.
**Namespaces:**
* `ui:` is registry-validated and reserved for built-in UI behavior.
* `app:` is user-defined/application-specific. It accepts any function name and any key-value parameter map, while still requiring valid action syntax (`namespace:function(key="value")`).

**Built-in Actions:**
* `ui:navigate(target="id")`
* `ui:toggle(id="target_id")`
* `ui:close()`

## 4. JSON Output Contract
The compiler must produce a JSON object with:
1. `tree`: The nested component AST.
2. `index`: A map of `id` to an object containing `path` and `type`.
3. `theme`: A flat map of thematic tokens, where `key = Color.key` and `value = Color.value`.
4. `metadata`: Compilation timestamp and compiler version.

Action values are sub-parsed and emitted inline at `tree.attributes.action` rather than as a separate top-level collection. `Theme` is treated as a top-level metadata definition and is not included in the UI `tree`.
