# CoreUI Components Reference

**Registry Version:** 1.2.0
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



