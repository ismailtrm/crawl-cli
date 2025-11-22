# crawl-cli

A lightweight, intelligent, and recursive web crawler CLI tool written in Go. 

It features optional **LLM integration (Gemini 2.5 Flash)** to analyze page content, score relevance based on your specific purpose, and intelligently prioritize which links to crawl next.

## Features

- **Recursive Crawling:** Configurable depth traversal.
- **LLM-Powered Analysis:** Uses Gemini to extract topics, summarize content, and score relevance.
- **Intelligent Navigation:** Prioritizes links that match your crawl purpose (when LLM is enabled).
- **Fast & Efficient:** Built with Go for high performance.

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

## Usage

### Basic Crawl
Crawl a single page and extract links/images.

```bash
./crawl-cli --link "https://example.com"
```

### Recursive Crawl
Crawl the page and follow links up to a depth of 2.

```bash
./crawl-cli --link "https://example.com" --depth 2
```

### Intelligent LLM Crawl
Use Gemini to analyze content and find specific information.

**Prerequisite:** Set your API key.
```bash
export GEMINI_API_KEY="your-api-key"
```

**Command:**
```bash
./crawl-cli --link "https://techcrunch.com" \
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
| `--llm` | Enable Gemini 2.5 Flash integration | false |
| `--purpose` | Describe goal for LLM to score relevance | "" |
| `--verbose` | Show detailed logs and content previews | false |
| `--timeout` | HTTP request timeout | 30s |
| `--insecure` | Skip TLS verification | false |

## License

MIT

```