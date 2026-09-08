# Documentation site

This directory contains the bilingual OINK/Hugo documentation site for
`sseserver-fiber`. It is a separate Go module so the Hugo theme dependency does
not affect the library module at the repository root.

## Requirements

- Git
- Go 1.27 or newer
- Hugo Extended 0.165.0 or newer (OINK supports 0.160.1+, while the Starter
  baseline is tested with 0.165.0)

Confirm that `hugo version` contains `extended`.

## Preview

```bash
cd site
hugo server
```

Open <http://localhost:1313/>. The Chinese site is available at
<http://localhost:1313/zh/>.

## Production build

```bash
cd site
hugo --cleanDestinationDir --gc --minify --environment production \
  --printPathWarnings --panicOnWarning
```

The generated site is written to `site/public/` and is intentionally ignored by
Git.
