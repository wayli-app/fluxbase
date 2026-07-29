---
editUrl: false
next: false
prev: false
title: "SubmitJobRequest"
---

Request to submit a new job

## Properties

| Property | Type | Description |
| ------ | ------ | ------ |
| <a id="job_name"></a> `job_name` | `string` | - |
| <a id="namespace"></a> `namespace?` | `string` | - |
| <a id="on_behalf_of"></a> `on_behalf_of?` | `OnBehalfOf` | Submit job on behalf of another user. Only available when using service_role authentication. The job will be created with the specified user's identity, allowing them to see the job and its logs via RLS. |
| <a id="payload"></a> `payload?` | `unknown` | - |
| <a id="priority"></a> `priority?` | `number` | - |
| <a id="scheduled"></a> `scheduled?` | `string` | - |
