# LLM Crawler - Usage Examples

## ✅ Installation Complete!

The LLM crawler has been successfully installed and is available globally as `crawl`.

## 🚀 Quick Start

### Basic Web Crawling

```bash
# Single page crawl
crawl --link 'https://example.com'

# Single page with verbose output
crawl --link 'https://example.com' --verbose

# Recursive crawl (page + its links)
crawl --link 'https://news.ycombinator.com' --depth 2 --verbose

# Custom timeout for slow sites
crawl --link 'https://slow-website.com' --timeout 60s --verbose
```

### AI-Powered Crawling (Requires Gemini API Key)

First, get your API key from: https://makersuite.google.com/app/apikey

```bash
# Set your API key (do this once per session)
export GEMINI_API_KEY='your-api-key-here'

# Basic AI crawl
crawl --link 'https://techcrunch.com' --llm --verbose

# Purpose-driven research crawl
crawl --link 'https://arxiv.org' --llm \
  --purpose 'find machine learning and AI research papers' \
  --depth 2 --verbose

# News discovery crawl
crawl --link 'https://news.ycombinator.com' --llm \
  --purpose 'discover startup and entrepreneurship content' \
  --depth 1 --verbose

# E-commerce product research
crawl --link 'https://shopping-site.com' --llm \
  --purpose 'find product reviews and pricing information' \
  --depth 2 --verbose
```

## 📊 Command Options

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--link` | string | *required* | URL to crawl |
| `--depth` | int | `0` | Maximum crawl depth (0=single page, 1=page+links, etc.) |
| `--timeout` | duration | `30s` | HTTP request timeout |
| `--verbose` | bool | `false` | Enable detailed output |
| `--llm` | bool | `false` | Enable AI-powered crawling with Gemini |
| `--purpose` | string | `""` | Describe crawl purpose for better AI decisions |
| `--help` | bool | `false` | Show help message |

## 🧠 AI Features (when using --llm)

- **Content Analysis**: AI analyzes page content for relevance and topics
- **Smart Link Selection**: AI suggests the most promising links to crawl next
- **Relevance Scoring**: Each page gets a 1-10 relevance score
- **Topic Extraction**: Automatically identifies main themes and topics
- **Content Summarization**: Generates concise page summaries
- **Quality Filtering**: Skip low-relevance pages automatically

## 📍 Project Location

- **Source code**: `/home/ismail/projects/crawler/`
- **Executable**: `crawl` (available globally)
- **Go binary location**: `$GOPATH/bin/crawl` (`/home/ismail/go/bin/crawl`)

## 🔧 Development

To rebuild after making changes to the source code:

```bash
cd /home/ismail/projects/crawler
go build -o crawl main.go
cp crawl $GOPATH/bin/  # Make it globally available
```

## 📚 Documentation

See `crawler_readme.md` for complete documentation and advanced usage examples.

## ⚠️ Important Notes

- **Rate Limiting**: No built-in rate limiting - be respectful to target sites
- **Memory Usage**: Large crawls may consume significant memory
- **AI Costs**: Gemini API usage incurs costs - monitor usage for large crawls
- **Legal & Ethical**: Respect robots.txt files and website terms of service