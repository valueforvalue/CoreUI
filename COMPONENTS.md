# CoreUI Components Reference

**Registry Version:** 1.3.0
**Schema Compatibility:** 1.0
**Last Updated:** 2026-04-26

## Global Actions

CoreUI action values use the form `namespace:function(key="value")`.

- `ui:` is **strictly validated** against the built-in UI action registry.
- `app:` is **user-defined/application-specific** and passes through as long as it follows valid action syntax.


## Box

**HasChildren:** true

| Prop | Type | Requirement |
| --- | --- | --- |
| background | String | Optional |
| border | Int | Optional |
| hidden | Bool | Optional |
| id | String | Required |
| padding | Unit | Optional |
| style | String | Optional |
| variant | String | Optional |



## Color

**HasChildren:** false

| Prop | Type | Requirement |
| --- | --- | --- |
| key | String | Required |
| value | String | Required |



## DataTable

**HasChildren:** false

| Prop | Type | Requirement |
| --- | --- | --- |
| hidden | Bool | Optional |
| id | String | Required |
| selectable | Bool | Optional |
| source | String | Optional |
| style | String | Optional |



## Graph

**HasChildren:** false

| Prop | Type | Requirement |
| --- | --- | --- |
| color | Theme Token | Optional |
| data | JSON Array or app:reference | Required |
| height | Unit | Optional |
| hidden | Bool | Optional |
| id | String | Required |
| labels | []string | Optional |
| style | String | Optional |
| type | String | Required |


### Best Practices

Use `labels` and `data` as parallel arrays with the same length so each label maps to the value at the same index.

- Prefer a quoted app reference such as `data="app:simulation.pressure_series"` when live data comes from your application layer.
- Use `type="line"` or `type="area"` for time series, `type="bar"` for ranked comparisons, and `type="pie"` only for small part-to-whole slices.
- Keep `height` unit-backed (for example `220px` or `40%`) so the compiler can reject invalid units before rendering.

~~~cui
Graph(
    id="throughput_graph",
    type="line",
    color="primary",
    height=240px,
    labels=["08:00", "10:00", "12:00", "14:00"],
    data=[18, 24, 21, 29]
)
~~~


## Grid

**HasChildren:** true

| Prop | Type | Requirement |
| --- | --- | --- |
| cols | Unit Array | Optional |
| gap | Unit | Optional |
| hidden | Bool | Optional |
| id | String | Required |
| rows | Unit Array | Optional |
| style | String | Optional |



## Image

**HasChildren:** false

| Prop | Type | Requirement |
| --- | --- | --- |
| alt | String | Optional |
| hidden | Bool | Optional |
| id | String | Required |
| src | String | Required |
| style | String | Optional |
| width | Unit | Optional |



## Input

**HasChildren:** false

| Prop | Type | Requirement |
| --- | --- | --- |
| bind | String | Optional |
| hidden | Bool | Optional |
| id | String | Required |
| label | String | Optional |
| style | String | Optional |
| type | String | Optional |



## Stack

**HasChildren:** true

| Prop | Type | Requirement |
| --- | --- | --- |
| align | String | Optional |
| dir | String | Optional |
| gap | Unit | Optional |
| hidden | Bool | Optional |
| id | String | Required |
| style | String | Optional |



## Text

**HasChildren:** false

| Prop | Type | Requirement |
| --- | --- | --- |
| hidden | Bool | Optional |
| id | String | Required |
| size | Unit | Optional |
| style | String | Optional |
| value | String | Optional |
| weight | String | Optional |



## Theme

**HasChildren:** true

| Prop | Type | Requirement |
| --- | --- | --- |
| hidden | Bool | Optional |
| id | String | Required |
| style | String | Optional |



## Trigger

**HasChildren:** false

| Prop | Type | Requirement |
| --- | --- | --- |
| action | Action | Optional |
| hidden | Bool | Optional |
| id | String | Required |
| label | String | Optional |
| style | String | Optional |
| variant | String | Optional |



## View

**HasChildren:** true

| Prop | Type | Requirement |
| --- | --- | --- |
| hidden | Bool | Optional |
| id | String | Required |
| style | String | Optional |
| theme | String | Optional |
| title | String | Optional |



