# CoreUI Technical Specification v1.0

## 1. BNF Grammar
<program>    ::= <component>
<component>  ::= <type> "(" <attr_list> ")" [ "{" <children> "}" ]
<type>       ::= [A-Z][a-zA-Z0-9]*
<attr_list>  ::= <attr> { "," <attr> } | ""
<attr>       ::= "action" "=" <action_val> | <key> "=" <value>
<key>        ::= [a-z][a-z0-9_]*
<value>      ::= <string> | <unit> | <number> | <boolean> | <array>
<unit>       ::= <number> ("px" | "%" | "*" | "auto")
<action_val> ::= <namespace> ":" <call> [ "(" <param_list> ")" ]

## 2. Component Registry
Every component must support `id` (String - Mandatory), `hidden` (Bool), and `style` (String).

| Component | Allowed Attributes |
| :--- | :--- |
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
**Built-in Actions:**
* `ui:navigate(target=id)`
* `ui:toggle(id=target_id)`
* `ui:close()`

## 4. JSON Output Contract
The compiler must produce a JSON object with:
1. `tree`: The nested component AST.
2. `index`: A map of `id` to an object containing `path` and `type`.
3. `metadata`: Compilation timestamp and compiler version.

Action values are sub-parsed and emitted inline at `tree.attributes.action` rather than as a separate top-level collection.
