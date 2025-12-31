---
editUrl: false
next: false
prev: false
title: "TransformOptions"
---

Options for on-the-fly image transformations
Applied to storage downloads via query parameters

## Properties

| Property | Type | Description |
| ------ | ------ | ------ |
| `fit?` | [`ImageFitMode`](/api/sdk-react/type-aliases/imagefitmode/) | How to fit the image within target dimensions (default: cover) |
| `format?` | [`ImageFormat`](/api/sdk-react/type-aliases/imageformat/) | Output format (defaults to original format) |
| `height?` | `number` | Target height in pixels (0 or undefined = auto based on width) |
| `quality?` | `number` | Output quality 1-100 (default: 80) |
| `width?` | `number` | Target width in pixels (0 or undefined = auto based on height) |
