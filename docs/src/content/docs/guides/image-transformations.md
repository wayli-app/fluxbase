---
title: "Image Transformations"
---

Fluxbase provides on-the-fly image transformation for files stored in Storage buckets. Transform images by adding query parameters to download URLs - resize, crop, convert formats, and adjust quality without modifying original files.

## Overview

Image transformations are applied at download time:

- **Resize** - Scale images to specific dimensions
- **Crop** - Extract portions of images with different fit modes
- **Format conversion** - Convert between JPEG, PNG, WebP, and AVIF
- **Quality adjustment** - Control output file size vs quality

Original files remain unchanged. Transformed images can be cached for performance.

## Query Parameters

Add these parameters to any storage file URL:

| Parameter | Alias | Description | Example |
|-----------|-------|-------------|---------|
| `width` | `w` | Target width in pixels | `w=300` |
| `height` | `h` | Target height in pixels | `h=200` |
| `format` | `fmt` | Output format | `fmt=webp` |
| `quality` | `q` | Quality 1-100 (default: 80) | `q=85` |
| `fit` | - | Fit mode (see below) | `fit=cover` |

### Examples

```
# Resize to 300x200
/api/v1/storage/images/photo.jpg?w=300&h=200

# Resize width only (height scales proportionally)
/api/v1/storage/images/photo.jpg?w=300

# Convert to WebP
/api/v1/storage/images/photo.jpg?fmt=webp

# High quality WebP thumbnail
/api/v1/storage/images/photo.jpg?w=150&fmt=webp&q=90

# All options combined
/api/v1/storage/images/photo.jpg?w=300&h=200&fmt=webp&q=85&fit=cover
```

## Fit Modes

The `fit` parameter controls how images are resized to match target dimensions:

| Mode | Description | Use Case |
|------|-------------|----------|
| `cover` | Fill target dimensions, crop excess | Thumbnails, profile pictures |
| `contain` | Fit within dimensions, letterbox | Product images, galleries |
| `fill` | Stretch to fill exactly (may distort) | Backgrounds |
| `inside` | Scale down only, never up | Ensure images aren't enlarged |
| `outside` | Scale to be at least target size | Cover backgrounds |

### Visual Examples

**Original image: 800×600**

```
# cover (default) - fills 200x200, crops excess
?w=200&h=200&fit=cover
Result: 200×200 (cropped from center)

# contain - fits within 200x200, maintains aspect ratio
?w=200&h=200&fit=contain
Result: 200×150 (with letterboxing if background added)

# fill - stretches to exactly 200x200
?w=200&h=200&fit=fill
Result: 200×200 (may appear stretched)

# inside - scales down to fit, won't scale up
?w=1000&h=1000&fit=inside
Result: 800×600 (unchanged, already fits)

# outside - scales to cover target, may exceed
?w=200&h=200&fit=outside
Result: 267×200 (scaled to cover, not cropped)
```

## Supported Formats

### Input Formats

| Format | MIME Type |
|--------|-----------|
| JPEG | `image/jpeg` |
| PNG | `image/png` |
| WebP | `image/webp` |
| GIF | `image/gif` |
| TIFF | `image/tiff` |
| BMP | `image/bmp` |
| AVIF | `image/avif` |
| SVG | `image/svg+xml` |

### Output Formats

| Format | Parameter | Best For |
|--------|-----------|----------|
| WebP | `fmt=webp` | Modern browsers, best compression |
| JPEG | `fmt=jpg` | Photos, compatibility |
| PNG | `fmt=png` | Transparency, graphics |
| AVIF | `fmt=avif` | Best compression, modern browsers |

## SDK Usage

### TypeScript SDK

```typescript
import { FluxbaseClient } from '@fluxbase/sdk'

const client = new FluxbaseClient({ url: 'http://localhost:8080' })
const storage = client.storage

// Get transform URL (synchronous)
const url = storage.from('images').getTransformUrl('photo.jpg', {
  width: 300,
  height: 200,
  format: 'webp',
  quality: 85,
  fit: 'cover'
})
// Result: http://localhost:8080/api/v1/storage/images/photo.jpg?w=300&h=200&fmt=webp&q=85&fit=cover

// Download transformed image
const { data, error } = await storage
  .from('images')
  .download('photo.jpg', {
    transform: {
      width: 300,
      height: 200,
      format: 'webp'
    }
  })

// Create signed URL with transforms
const { data: signedUrl } = await storage
  .from('images')
  .createSignedUrl('photo.jpg', {
    expiresIn: 3600, // 1 hour
    transform: {
      width: 400,
      format: 'webp'
    }
  })
```

### React SDK

```tsx
import {
  useStorageTransformUrl,
  useStorageSignedUrl
} from '@fluxbase/sdk-react'

function ImageGallery() {
  // Get transform URL (synchronous, no network request)
  const thumbnailUrl = useStorageTransformUrl('images', 'photo.jpg', {
    width: 150,
    height: 150,
    format: 'webp',
    fit: 'cover'
  })

  return <img src={thumbnailUrl} alt="Thumbnail" />
}

function ProtectedImage() {
  // Get signed URL with transforms
  const { data: signedUrl, isLoading } = useStorageSignedUrl(
    'private-images',
    'photo.jpg',
    {
      expiresIn: 3600,
      transform: {
        width: 800,
        format: 'webp',
        quality: 90
      }
    }
  )

  if (isLoading) return <div>Loading...</div>

  return <img src={signedUrl} alt="Protected" />
}
```

### Responsive Images

Generate multiple sizes for responsive images:

```tsx
function ResponsiveImage({ path }: { path: string }) {
  const storage = useStorage()

  const srcSet = [
    { width: 320, descriptor: '320w' },
    { width: 640, descriptor: '640w' },
    { width: 1280, descriptor: '1280w' },
  ].map(({ width, descriptor }) => {
    const url = storage.from('images').getTransformUrl(path, {
      width,
      format: 'webp'
    })
    return `${url} ${descriptor}`
  }).join(', ')

  return (
    <img
      srcSet={srcSet}
      sizes="(max-width: 640px) 320px, (max-width: 1280px) 640px, 1280px"
      src={storage.from('images').getTransformUrl(path, { width: 640, format: 'webp' })}
      alt=""
    />
  )
}
```

## Signed URLs with Transforms

Transform parameters are included in the signed URL signature, preventing tampering:

```typescript
// Create signed URL with transforms
const { data } = await storage
  .from('private-bucket')
  .createSignedUrl('image.jpg', {
    expiresIn: 3600,
    transform: {
      width: 400,
      height: 300,
      format: 'webp'
    }
  })

// Signed URL includes transform params in signature
// Modifying ?w=400 to ?w=800 will invalidate the signature
```

## Configuration

Configure image transformations in your Fluxbase config:

```yaml
storage:
  transforms:
    enabled: true
    default_quality: 80      # Default quality when not specified
    max_width: 4096         # Maximum allowed width
    max_height: 4096        # Maximum allowed height
    allowed_formats:        # Allowed output formats
      - webp
      - jpg
      - png
      - avif
    cache_ttl: 86400        # Cache duration in seconds (24 hours)
```

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `FLUXBASE_STORAGE_TRANSFORMS_ENABLED` | Enable transforms | `true` |
| `FLUXBASE_STORAGE_TRANSFORMS_DEFAULT_QUALITY` | Default quality | `80` |
| `FLUXBASE_STORAGE_TRANSFORMS_MAX_WIDTH` | Max width | `4096` |
| `FLUXBASE_STORAGE_TRANSFORMS_MAX_HEIGHT` | Max height | `4096` |
| `FLUXBASE_STORAGE_TRANSFORMS_MAX_TOTAL_PIXELS` | Max total pixels | `16000000` |
| `FLUXBASE_STORAGE_TRANSFORMS_BUCKET_SIZE` | Dimension rounding | `50` |
| `FLUXBASE_STORAGE_TRANSFORMS_RATE_LIMIT` | Transforms/min/user | `60` |
| `FLUXBASE_STORAGE_TRANSFORMS_TIMEOUT` | Max transform duration | `30s` |
| `FLUXBASE_STORAGE_TRANSFORMS_MAX_CONCURRENT` | Max concurrent transforms | `4` |
| `FLUXBASE_STORAGE_TRANSFORMS_CACHE_ENABLED` | Enable caching | `true` |
| `FLUXBASE_STORAGE_TRANSFORMS_CACHE_TTL` | Cache TTL | `24h` |
| `FLUXBASE_STORAGE_TRANSFORMS_CACHE_MAX_SIZE` | Max cache size | `1073741824` |

## Performance & Caching

### Caching Strategy

Transformed images are cached in an internal `_transform_cache` bucket to avoid reprocessing:

1. **First request** - Image is transformed and stored in cache
2. **Subsequent requests** - Cached version is served instantly
3. **Cache eviction** - LRU (Least Recently Used) entries are evicted when cache reaches max size
4. **TTL expiration** - Cached transforms expire after the configured TTL (default: 24 hours)

### Cache Key Format

```
sha256({bucket}/{path}:{width}:{height}:{format}:{quality}:{fit})
```

The cache key is a SHA256 hash of the transform parameters, ensuring unique cache entries for each variation.

### Dimension Bucketing

To reduce cache fragmentation and improve hit rates, dimensions are rounded to the nearest bucket size (default: 50px):

- Request: `?w=147&h=203` → Actual: `?w=150&h=200`
- Request: `?w=320&h=240` → Actual: `?w=300&h=250`

This prevents attackers from busting the cache with many slightly different dimension requests.

### Best Practices

1. **Use WebP** - Best compression for modern browsers
2. **Specify dimensions** - Prevents unnecessary large transforms
3. **Use signed URLs** - For private buckets and CDN caching
4. **Set appropriate cache headers** - Let browsers and CDNs cache

```typescript
// Good: Specific dimensions and format
storage.from('images').getTransformUrl('photo.jpg', {
  width: 300,
  height: 200,
  format: 'webp'
})

// Avoid: Only format conversion (transforms entire image)
storage.from('images').getTransformUrl('photo.jpg', {
  format: 'webp'  // Still transforms full-size image
})
```

## REST API

### Transform via Query Parameters

```bash
# Basic resize
curl "http://localhost:8080/api/v1/storage/images/photo.jpg?w=300&h=200"

# Format conversion
curl "http://localhost:8080/api/v1/storage/images/photo.jpg?fmt=webp"

# All options
curl "http://localhost:8080/api/v1/storage/images/photo.jpg?w=300&h=200&fmt=webp&q=85&fit=cover"
```

### With Authentication

```bash
curl "http://localhost:8080/api/v1/storage/private-bucket/photo.jpg?w=300&fmt=webp" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### Signed URL with Transforms

```bash
# Create signed URL (via API)
curl -X POST "http://localhost:8080/api/v1/storage/images/sign/photo.jpg" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "expires_in": 3600,
    "transform": {
      "width": 400,
      "format": "webp"
    }
  }'

# Response
{
  "signed_url": "http://localhost:8080/api/v1/storage/images/photo.jpg?w=400&fmt=webp&token=xxx&expires=xxx"
}
```

## Troubleshooting

### Transform returns original image

- Verify the file is an image (check Content-Type)
- Ensure transforms are enabled in config
- Check that requested dimensions are within limits

### Poor quality output

- Increase `quality` parameter (default is 80)
- Use appropriate format (JPEG for photos, PNG for graphics)
- Avoid upscaling (use `fit=inside` to prevent)

### Slow transformations

- First requests are slower (cache miss)
- Large source images take longer
- Consider pre-generating common sizes

### AVIF not working

- AVIF requires libvips with AVIF support
- Check server logs for codec errors
- Fall back to WebP for broad compatibility

### Out of memory errors

- Reduce max dimensions in config
- Process large images in batches
- Increase server memory allocation

## Limits

| Limit | Default | Configurable |
|-------|---------|--------------|
| Max width | 4096px | Yes |
| Max height | 4096px | Yes |
| Max total pixels | 16M | Yes |
| Max input size | Server memory | No |
| Rate limit | 60/min/user | Yes |
| Concurrent transforms | 4 | Yes |
| Supported formats | See above | Partial |

## Security

Fluxbase implements several security measures to protect against abuse:

### Resource Protection

- **Dimension limits** - Prevents excessively large output images
- **Total pixel limit** - Prevents memory exhaustion (default: 16 megapixels)
- **Rate limiting** - Limits transforms per user per minute (default: 60)
- **Concurrency limit** - Limits simultaneous transforms (default: 4)

### Cache Security

- **Dimension bucketing** - Rounds dimensions to nearest 50px to reduce cache-busting attacks
- **LRU eviction** - Automatically removes oldest entries when cache is full
- **TTL expiration** - Cached transforms expire after 24 hours by default

### Path Security

- **Directory traversal protection** - All file paths are validated and sanitized
- **RLS enforcement** - Storage permissions are checked before any transform

## Next Steps

- [Storage Guide](/docs/guides/storage) - Storage bucket management
- [Row-Level Security](/docs/guides/row-level-security) - Secure file access
- [Edge Functions](/docs/guides/edge-functions) - Custom processing logic
