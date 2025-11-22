# crawl-cli

A lightweight, intelligent, and recursive web crawler CLI tool written in Go.

It features optional **LLM integration (Gemini 2.5 Flash)** to analyze page content, score relevance, and intelligently prioritize links. It also supports **JavaScript rendering** via a self-hosted Browserless instance.

## Features

- **Recursive Crawling:** Configurable depth traversal.
- **JavaScript Rendering:** Supports SPA crawling using a self-hosted [Browserless](https://www.browserless.io/) instance (REST API).
- **LLM-Powered Analysis:** Uses Gemini to extract topics, summarize content, and score relevance.
- **Scrape Mode:** Output raw rendered HTML directly to stdout for piping to other tools.
- **Fast & Efficient:** Built with Go.

## Installation

1. **Clone the repository:**
   ```bash
   git clone https://github.com/yourusername/crawl-cli.git
   cd crawl-cli
   ```

2. **Build the binary:**
   ```bash
   go build -o crawl-cli
   ```

3. **(Optional) Install to path:**
   ```bash
   go install .
   ```

## Usage

### Basic Crawl
Crawl a single static page.
```bash
crawl-cli --link "https://example.com"
```

### Javascript Rendering (SPA Support)
To crawl Single Page Applications (React, Vue, etc.), you can connect `crawl-cli` to a Browserless instance.

*This tool was tested with a self-hosted Browserless instance running on a VPS via [Dokploy](https://dokploy.com/).*

```bash
export BROWSERLESS_TOKEN="your-secure-token"

crawl-cli --link "https://spa-website.com" \
  --browserless "https://browserless.your-domain.com"
```

### Scrape Mode (Raw Output)
Fetch the rendered HTML and output it directly to stdout (useful for piping).

```bash
crawl-cli --link "https://spa-website.com" \
  --browserless "https://browserless.your-domain.com" \
  --scrape > output.html
```

### Intelligent LLM Crawl
Use Gemini to analyze content and find specific information.

**Prerequisite:**
```bash
export GEMINI_API_KEY="your-gemini-key"
```

**Command:**
```bash
crawl-cli --link "https://techcrunch.com" \
  --llm \
  --purpose "find news about artificial intelligence startups" \
  --depth 2 \
  --verbose
```

## Options

| Flag | Description | Default | 
|------|-------------|---------|
| `--link` | The target URL to crawl (required) | | 
| `--depth` | Crawl depth (0=single page, 1=links on page, etc) | 0 | 
| `--scrape` | Output raw HTML to stdout and exit | false |
| `--browserless` | URL of your Browserless instance (can also use `BROWSERLESS_URL` env var) | "" |
| `--token` | Browserless API Token (can also use `BROWSERLESS_TOKEN` env var) | "" || `--llm` | Enable Gemini 2.5 Flash integration | false | 
| `--purpose` | Describe goal for LLM to score relevance | "" | 
| `--verbose` | Show detailed logs and content previews | false | 
| `--timeout` | HTTP request timeout | 30s | 
| `--insecure` | Skip TLS verification | false | 

## Self-Hosting Browserless

You can easily host your own Browserless instance using Docker or Dokploy.

**Docker Compose Example:**
```yaml
services:
  browserless:
    image: browserless/chrome:latest
    ports:
      - "3000:3000"
    environment:
      - TOKEN=your-secure-token
```

## License

MIT
