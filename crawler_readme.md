# 🕷️ AI-Enhanced Web Crawler

A powerful command-line web crawler built in Go that combines traditional web scraping with AI-powered content analysis using Google's Gemini 2.5 Flash.

## ✨ Features

### 🔍 Traditional Web Crawling
- **Single & Recursive Crawling**: Crawl individual pages or follow links to specified depths
- **Content Extraction**: Extracts titles, links, images, and text content from web pages
- **Smart URL Handling**: Resolves relative URLs to absolute, handles redirects
- **Domain Restriction**: Stays within the same domain during recursive crawling
- **Duplicate Detection**: Avoids visiting the same URL twice
- **Configurable Timeouts**: Set custom HTTP request timeouts
- **Verbose Logging**: Detailed output for debugging and monitoring

### 🤖 AI-Powered Intelligence (Optional)
- **Content Analysis**: AI analyzes page content for relevance and topics
- **Smart Link Selection**: AI suggests the most promising links to crawl next
- **Relevance Scoring**: Each page gets a 1-10 relevance score
- **Topic Extraction**: Automatically identifies main themes and topics
- **Content Summarization**: Generates concise page summaries
- **Purpose-Driven Crawling**: Tailor crawls to specific research goals
- **Quality Filtering**: Skip low-relevance pages automatically

## 📦 Installation

### Prerequisites
- Go 1.19 or higher
- (Optional) Google Gemini API key for AI features

### Setup
1. **Clone and build**:
```bash
git clone <repository-url>
cd web-crawler
go mod init llm-crawler
go get golang.org/x/net/html
go build -o crawl main.go
```

2. **Make globally available** (optional):
```bash
sudo mv crawl /usr/local/bin/
# or add to your PATH
```

3. **Setup AI features** (optional):
```bash
# Get API key from: https://makersuite.google.com/app/apikey
export GEMINI_API_KEY='your-api-key-here'
```

## 🚀 Usage

### Basic Syntax
```bash
./crawl --link <URL> [OPTIONS]
```

### Command-Line Options

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--link` | string | *required* | URL to crawl |
| `--depth` | int | `0` | Maximum crawl depth (0=single page, 1=page+links, etc.) |
| `--timeout` | duration | `30s` | HTTP request timeout |
| `--verbose` | bool | `false` | Enable detailed output |
| `--llm` | bool | `false` | Enable AI-powered crawling with Gemini |
| `--purpose` | string | `""` | Describe crawl purpose for better AI decisions |
| `--help` | bool | `false` | Show help message |

## 📚 Examples

### Traditional Web Crawling (No AI)

#### Single Page Crawl
```bash
./crawl --link 'https://example.com'
```

#### Recursive Crawl with Depth
```bash
./crawl --link 'https://news.ycombinator.com' --depth 2 --verbose
```

#### Custom Timeout
```bash
./crawl --link 'https://slow-website.com' --timeout 60s --verbose
```

### AI-Powered Crawling

#### Basic AI Crawl
```bash
export GEMINI_API_KEY='your-api-key'
./crawl --link 'https://techcrunch.com' --llm --verbose
```

#### Purpose-Driven Research Crawl
```bash
./crawl --link 'https://arxiv.org' --llm \
  --purpose 'find machine learning and AI research papers' \
  --depth 2 --verbose
```

#### News Discovery Crawl
```bash
./crawl --link 'https://news.ycombinator.com' --llm \
  --purpose 'discover startup and entrepreneurship content' \
  --depth 1
```

#### E-commerce Product Research
```bash
./crawl --link 'https://shopping-site.com' --llm \
  --purpose 'find product reviews and pricing information' \
  --depth 2 --verbose
```

## 📊 Output Format

### Traditional Crawling Output
```
================================================================================
URL: https://example.com
Status: 200
Content-Type: text/html; charset=UTF-8
Title: Example Domain

Found 5 links:
  - https://example.com/about
  - https://example.com/contact
  ...

Found 3 images:
  - https://example.com/logo.png
  ...

Text content preview:
This domain is for use in illustrative examples...
```

### AI-Enhanced Output
```
================================================================================
URL: https://techcrunch.com/ai-news
Status: 200
Content-Type: text/html; charset=UTF-8
Title: Latest AI News and Updates
🎯 Relevance Score: 9/10
🏷️ Topics: artificial intelligence, machine learning, startups, technology
📝 Summary: Latest developments in AI technology including new model releases...

Found 25 links:
  - https://techcrunch.com/ai-funding
  ...

🤖 LLM Suggested Priority Links (3):
  ⭐ https://techcrunch.com/openai-announcement
  ⭐ https://techcrunch.com/ai-startup-funding
  ⭐ https://techcrunch.com/machine-learning-breakthrough
```

### Crawl Summary (Multiple Pages)
```
🧠 LLM Insights Summary:
   Average Relevance: 8/10
   Unique Topics Found: 12
   Most Common Topics: artificial intelligence (5), startups (4), funding (3)

✅ Crawl completed. Visited 15 pages.
```

## ⚙️ Configuration

### Environment Variables
- `GEMINI_API_KEY`: Required for AI-powered features
  - Get your key at: https://makersuite.google.com/app/apikey

### Crawling Behavior

#### Traditional Mode (without `--llm`)
- Follows ALL links found on pages
- No content filtering or analysis
- Crawls until max depth reached
- No external API calls

#### AI Mode (with `--llm`)
- Analyzes content relevance before crawling deeper
- Uses AI-suggested priority links instead of all links
- Filters out low-relevance pages (< 5/10 score)
- Provides content insights and summaries

## 🎯 Use Cases

### Content Research
```bash
./crawl --link 'https://research-site.com' --llm \
  --purpose 'find academic papers on climate change' --depth 3
```

### Competitive Analysis
```bash
./crawl --link 'https://competitor.com' --llm \
  --purpose 'analyze product features and pricing' --depth 2
```

### News Monitoring
```bash
./crawl --link 'https://news-site.com' --llm \
  --purpose 'track technology and startup news' --depth 1
```

### SEO Analysis
```bash
./crawl --link 'https://my-website.com' \
  --depth 3 --verbose  # Traditional crawl for complete site map
```

## 🔧 Technical Details

### Supported Content Types
- HTML pages (text/html)
- Follows HTTP redirects
- Handles relative and absolute URLs
- Respects domain boundaries

### Data Extraction
- **Titles**: `<title>` tag content
- **Links**: All `<a href="">` attributes
- **Images**: All `<img src="">` attributes  
- **Text**: Content from `<p>`, `<div>`, `<span>`, `<h1-h6>`, `<article>`, `<section>` tags

### AI Analysis (Gemini 2.5 Flash)
- Content relevance scoring (1-10)
- Topic extraction and categorization
- Content summarization
- Priority link recommendation
- Purpose-driven content filtering

## 🚨 Limitations & Considerations

### Rate Limiting
- No built-in rate limiting (be respectful to target sites)
- Consider adding delays for large crawls
- AI API has usage limits and costs

### Memory Usage
- Stores visited URLs in memory
- Large crawls may consume significant memory
- Consider crawl depth for memory management

### AI Costs
- Gemini API usage incurs costs
- Each page analysis makes one API call
- Monitor usage for large crawls

### Legal & Ethical
- Respect robots.txt files (not automatically enforced)
- Follow website terms of service
- Don't overload target servers
- Consider privacy implications of AI content analysis

## 🔍 Troubleshooting

### Common Issues

#### "GEMINI_API_KEY environment variable is required"
```bash
export GEMINI_API_KEY='your-actual-api-key'
```

#### Timeout Errors
```bash
# Increase timeout for slow sites
./crawl --link 'https://slow-site.com' --timeout 60s
```

#### Too Many Links
```bash
# Use AI mode to focus on relevant content
./crawl --link 'https://big-site.com' --llm --purpose 'specific topic' --depth 1
```

#### Memory Issues
```bash
# Reduce crawl depth
./crawl --link 'https://huge-site.com' --depth 1 --verbose
```

## 📈 Performance Tips

1. **Use AI mode for focused crawling**: Better than crawling everything
2. **Start with depth 1**: Test before going deeper
3. **Use specific purposes**: More relevant AI suggestions
4. **Monitor API usage**: Check Gemini API costs
5. **Use verbose mode**: Debug crawling behavior

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make changes with tests
4. Submit a pull request

## 📄 License

MIT License - see LICENSE file for details

## 🆘 Support

- Check command help: `./crawl --help`
- Enable verbose logging: `--verbose`
- Review API key setup for AI features
- Monitor network connectivity and target site availability

---

**Built with ❤️ in Go | Powered by Google Gemini 2.5 Flash**