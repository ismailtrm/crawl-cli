package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// CrawlResult stores the results of crawling a page
type CrawlResult struct {
	URL         string
	Title       string
	Links       []string
	Text        string
	Images      []string
	StatusCode  int
	ContentType string
	Error       error
	// LLM-enhanced fields
	Summary        string   `json:"summary,omitempty"`
	Topics         []string `json:"topics,omitempty"`
	Relevance      int      `json:"relevance,omitempty"` // 1-10 score
	SuggestedLinks []string `json:"suggested_links,omitempty"`
}

// Crawler configuration
type Crawler struct {
	client       *http.Client
	maxDepth     int
	currentDepth int
	visited      map[string]bool
	baseURL      *url.URL
	verbose      bool
	llmEnabled   bool
	geminiAPIKey string
	crawlPurpose string
}

// GeminiRequest structure for API calls
type GeminiRequest struct {
	Contents []GeminiContent `json:"contents"`
}

type GeminiContent struct {
	Parts []GeminiPart `json:"parts"`
}

type GeminiPart struct {
	Text string `json:"text"`
}

// GeminiResponse structure
type GeminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

// LLMAnalysis structure for parsing LLM responses
type LLMAnalysis struct {
	Summary        string   `json:"summary"`
	Topics         []string `json:"topics"`
	Relevance      int      `json:"relevance"`
	SuggestedLinks []string `json:"suggested_links"`
	ShouldCrawl    bool     `json:"should_crawl"`
	Priority       int      `json:"priority"`
}

// NewCrawler creates a new crawler instance
func NewCrawler(timeout time.Duration, maxDepth int, verbose bool, llmEnabled bool, purpose string, insecure bool) *Crawler {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: insecure},
	}

	crawler := &Crawler{
		client: &http.Client{
			Timeout:   timeout,
			Transport: transport,
		},
		maxDepth:     maxDepth,
		visited:      make(map[string]bool),
		verbose:      verbose,
		llmEnabled:   llmEnabled,
		crawlPurpose: purpose,
	}

	if llmEnabled {
		crawler.geminiAPIKey = os.Getenv("GEMINI_API_KEY")
		if crawler.geminiAPIKey == "" {
			log.Fatal("GEMINI_API_KEY environment variable is required when using --llm flag")
		}
	}

	return crawler
}

// Crawl fetches and parses a single URL
func (c *Crawler) Crawl(targetURL string) (*CrawlResult, error) {
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	if c.baseURL == nil {
		c.baseURL = &url.URL{
			Scheme: parsedURL.Scheme,
			Host:   parsedURL.Host,
		}
	}

	if c.visited[targetURL] {
		if c.verbose {
			fmt.Printf("Skipping already visited: %s\n", targetURL)
		}
		return nil, fmt.Errorf("already visited")
	}
	c.visited[targetURL] = true

	if c.verbose {
		fmt.Printf("Crawling: %s\n", targetURL)
	}

	resp, err := c.client.Get(targetURL)
	if err != nil {
		return &CrawlResult{
			URL:   targetURL,
			Error: err,
		}, err
	}
	defer resp.Body.Close()

	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		return &CrawlResult{
			URL:         targetURL,
			StatusCode:  resp.StatusCode,
			ContentType: contentType,
		}, nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &CrawlResult{
			URL:        targetURL,
			StatusCode: resp.StatusCode,
			Error:      err,
		}, err
	}

	doc, err := html.Parse(strings.NewReader(string(body)))
	if err != nil {
		return &CrawlResult{
			URL:        targetURL,
			StatusCode: resp.StatusCode,
			Error:      err,
		}, err
	}

	result := &CrawlResult{
		URL:         targetURL,
		StatusCode:  resp.StatusCode,
		ContentType: contentType,
		Links:       []string{},
		Images:      []string{},
	}

	c.extractData(doc, result, parsedURL)

	if c.llmEnabled {
		err := c.enhanceWithLLM(result, string(body))
		if err != nil {
			if c.verbose {
				fmt.Printf("LLM analysis failed for %s: %v\n", targetURL, err)
			}
		}
	}

	return result, nil
}

func (c *Crawler) enhanceWithLLM(result *CrawlResult, htmlContent string) error {
	prompt := c.buildAnalysisPrompt(result, htmlContent)

	analysis, err := c.callGemini(prompt)
	if err != nil {
		return fmt.Errorf("failed to call Gemini API: %w", err)
	}

	llmResult, err := c.parseLLMResponse(analysis)
	if err != nil {
		return fmt.Errorf("failed to parse LLM response: %w", err)
	}

	result.Summary = llmResult.Summary
	result.Topics = llmResult.Topics
	result.Relevance = llmResult.Relevance
	result.SuggestedLinks = llmResult.SuggestedLinks

	if c.verbose {
		fmt.Printf("[LLM Analysis] %s\n", result.URL)
		fmt.Printf("   Relevance: %d/10\n", result.Relevance)
		fmt.Printf("   Topics: %s\n", strings.Join(result.Topics, ", "))
		fmt.Printf("   Summary: %s\n", result.Summary)
		if len(result.SuggestedLinks) > 0 {
			fmt.Printf("   Suggested priority links: %d\n", len(result.SuggestedLinks))
		}
	}

	return nil
}

func (c *Crawler) buildAnalysisPrompt(result *CrawlResult, htmlContent string) string {
	maxContent := 8000
	if len(htmlContent) > maxContent {
		htmlContent = htmlContent[:maxContent] + "..."
	}

	purpose := c.crawlPurpose
	if purpose == "" {
		purpose = "general web crawling and content discovery"
	}

	prompt := fmt.Sprintf(`You are an AI assistant helping with intelligent web crawling. 

CRAWL PURPOSE: %s

Analyze the following webpage and provide structured insights:

URL: %s
Title: %s
Text Content: %s

Based on this content, please provide a JSON response with the following structure:
{
  "summary": "Brief 2-3 sentence summary of the page content",
  "topics": ["topic1", "topic2", "topic3"],
  "relevance": 8,
  "suggested_links": ["url1", "url2"],
  "should_crawl": true,
  "priority": 7
}

Guidelines:
- relevance: Score 1-10 how relevant this page is to the crawl purpose
- topics: Extract 3-5 main topics/themes from the content
- suggested_links: From the links found, suggest up to 3 most promising ones to crawl next
- should_crawl: Whether this type of content is worth crawling
- priority: How important this page is (1-10, higher = more important)

Respond ONLY with valid JSON, no other text.`, purpose, result.URL, result.Title, result.Text[:min(2000, len(result.Text))])

	return prompt
}

func (c *Crawler) callGemini(prompt string) (string, error) {
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash:generateContent?key=%s", c.geminiAPIKey)

	request := GeminiRequest{
		Contents: []GeminiContent{
			{
				Parts: []GeminiPart{
					{Text: prompt},
				},
			},
		},
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", err
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API call failed with status %d: %s", resp.StatusCode, string(body))
	}

	var response GeminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", err
	}

	if len(response.Candidates) == 0 || len(response.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("empty response from Gemini")
	}

	return response.Candidates[0].Content.Parts[0].Text, nil
}

func (c *Crawler) parseLLMResponse(response string) (*LLMAnalysis, error) {
	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimSuffix(response, "```")
	response = strings.TrimSpace(response)

	var analysis LLMAnalysis
	err := json.Unmarshal([]byte(response), &analysis)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w, response was: %s", err, response)
	}

	return &analysis, nil
}

func (c *Crawler) extractData(n *html.Node, result *CrawlResult, baseURL *url.URL) {
	if n.Type == html.ElementNode {
		switch n.Data {
		case "title":
			if n.FirstChild != nil && n.FirstChild.Type == html.TextNode {
				result.Title = strings.TrimSpace(n.FirstChild.Data)
			}
		case "a":
			for _, attr := range n.Attr {
				if attr.Key == "href" {
					link := c.resolveURL(attr.Val, baseURL)
					if link != "" {
						result.Links = append(result.Links, link)
					}
				}
			}
		case "img":
			for _, attr := range n.Attr {
				if attr.Key == "src" {
					imgURL := c.resolveURL(attr.Val, baseURL)
					if imgURL != "" {
						result.Images = append(result.Images, imgURL)
					}
				}
			}
		case "p", "div", "span", "h1", "h2", "h3", "h4", "h5", "h6", "article", "section":
			text := c.extractText(n)
			if text != "" && len(text) > 10 {
				result.Text += text + "\n"
			}
		}
	}

	for child := n.FirstChild; child != nil; child = child.NextSibling {
		c.extractData(child, result, baseURL)
	}
}

func (c *Crawler) extractText(n *html.Node) string {
	var text string
	if n.Type == html.TextNode {
		text = strings.TrimSpace(n.Data)
	}
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		text += " " + c.extractText(child)
	}
	return strings.TrimSpace(text)
}

func (c *Crawler) resolveURL(href string, base *url.URL) string {
	u, err := url.Parse(href)
	if err != nil {
		return ""
	}

	if u.Scheme != "" && u.Scheme != "http" && u.Scheme != "https" {
		return ""
	}

	resolved := base.ResolveReference(u)
	return resolved.String()
}

func (c *Crawler) CrawlRecursive(targetURL string, depth int) ([]*CrawlResult, error) {
	if depth > c.maxDepth {
		return nil, nil
	}

	results := []*CrawlResult{}

	result, err := c.Crawl(targetURL)
	if err != nil && result == nil {
		return results, err
	}

	if result != nil {
		results = append(results, result)

		if depth < c.maxDepth && result.Links != nil {
			linksToProcess := result.Links
			if c.llmEnabled && len(result.SuggestedLinks) > 0 {
				linksToProcess = result.SuggestedLinks
				if c.verbose {
					fmt.Printf("[Info] Using LLM-suggested links (%d) instead of all links (%d)\n",
						len(result.SuggestedLinks), len(result.Links))
				}
			}

			for _, link := range linksToProcess {
				linkURL, err := url.Parse(link)
				if err != nil {
					continue
				}

				if linkURL.Host == c.baseURL.Host {
					if c.llmEnabled && result.Relevance < 5 {
						if c.verbose {
							fmt.Printf("[Skip] Low relevance page (score: %d): %s\n", result.Relevance, link)
						}
						continue
					}

					subResults, _ := c.CrawlRecursive(link, depth+1)
					results = append(results, subResults...)
				}
			}
		}
	}

	return results, nil
}

func PrintResult(result *CrawlResult, verbose bool, llmEnabled bool) {
	fmt.Printf("\n" + strings.Repeat("=", 80) + "\n")
	fmt.Printf("URL: %s\n", result.URL)
	fmt.Printf("Status: %d\n", result.StatusCode)
	fmt.Printf("Content-Type: %s\n", result.ContentType)

	if result.Error != nil {
		fmt.Printf("Error: %v\n", result.Error)
		return
	}

	if result.Title != "" {
		fmt.Printf("Title: %s\n", result.Title)
	}

	if llmEnabled {
		if result.Relevance > 0 {
			fmt.Printf("Relevance Score: %d/10\n", result.Relevance)
		}
		if len(result.Topics) > 0 {
			fmt.Printf("Topics: %s\n", strings.Join(result.Topics, ", "))
		}
		if result.Summary != "" {
			fmt.Printf("Summary: %s\n", result.Summary)
		}
	}

	if len(result.Links) > 0 {
		fmt.Printf("\nFound %d links:\n", len(result.Links))
		if verbose {
			displayCount := min(10, len(result.Links))
			for i := 0; i < displayCount; i++ {
				fmt.Printf("  - %s\n", result.Links[i])
			}
			if len(result.Links) > 10 {
				fmt.Printf("  ... and %d more\n", len(result.Links)-10)
			}
		}
	}

	if llmEnabled && len(result.SuggestedLinks) > 0 {
		fmt.Printf("\nLLM Suggested Priority Links (%d):\n", len(result.SuggestedLinks))
		if verbose {
			for _, link := range result.SuggestedLinks {
				fmt.Printf("  - %s\n", link)
			}
		}
	}

	if len(result.Images) > 0 {
		fmt.Printf("\nFound %d images:\n", len(result.Images))
		if verbose {
			displayCount := min(5, len(result.Images))
			for i := 0; i < displayCount; i++ {
				fmt.Printf("  - %s\n", result.Images[i])
			}
			if len(result.Images) > 5 {
				fmt.Printf("  ... and %d more\n", len(result.Images)-5)
			}
		}
	}

	if result.Text != "" && verbose {
		fmt.Printf("\nText content preview:\n")
		preview := result.Text
		if len(preview) > 500 {
			preview = preview[:500] + "..."
		}
		fmt.Printf("%s\n", preview)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func main() {
	var (
		link     = flag.String("link", "", "URL to crawl (required)")
		depth    = flag.Int("depth", 0, "Maximum crawl depth (0 = single page, 1 = page + its links, etc.)")
		timeout  = flag.Duration("timeout", 30*time.Second, "HTTP request timeout")
		verbose  = flag.Bool("verbose", false, "Enable verbose output")
		llm      = flag.Bool("llm", false, "Enable LLM-powered intelligent crawling with Gemini 2.5 Flash")
		purpose  = flag.String("purpose", "", "Describe the purpose of your crawl to help the LLM make better decisions")
		insecure = flag.Bool("insecure", false, "Skip TLS certificate verification for HTTPS sites")
		help     = flag.Bool("help", false, "Show help message")
	)

	flag.Parse()

	if *help || *link == "" {
		fmt.Println("AI-Enhanced Web Crawler CLI Tool")
		fmt.Println("\nUsage:")
		fmt.Printf("  %s --link <URL> [options]\n", os.Args[0])
		fmt.Println("\nOptions:")
		flag.PrintDefaults()
		fmt.Println("\nLLM Setup:")
		fmt.Println("  Set GEMINI_API_KEY environment variable to use --llm flag")
		fmt.Println("  Get your API key at: https://makersuite.google.com/app/apikey")
		fmt.Println("\nExamples:")
		fmt.Printf("  # Basic crawl\n")
		fmt.Printf("  %s --link 'https://example.com'\n", os.Args[0])
		fmt.Printf("\n  # LLM-powered crawl with purpose\n")
		fmt.Printf("  export GEMINI_API_KEY='your-api-key'\n")
		fmt.Printf("  %s --link 'https://techcrunch.com' --llm --purpose 'find AI and machine learning news' --depth 2 --verbose\n", os.Args[0])
		
		if *link == "" && !*help {
			fmt.Println("\nError: --link flag is required")
			os.Exit(1)
		}
		os.Exit(0)
	}

	if !strings.HasPrefix(*link, "http://") && !strings.HasPrefix(*link, "https://") {
		log.Fatal("Error: URL must start with http:// or https://")
	}

	if *llm {
		if os.Getenv("GEMINI_API_KEY") == "" {
			log.Fatal("Error: GEMINI_API_KEY environment variable is required when using --llm flag")
		}
		fmt.Println("LLM-powered crawling enabled with Gemini 2.5 Flash")
		if *purpose != "" {
			fmt.Printf("Crawl purpose: %s\n", *purpose)
		}
	}

	crawler := NewCrawler(*timeout, *depth, *verbose, *llm, *purpose, *insecure)

	fmt.Printf("Starting crawl of %s (depth=%d, timeout=%v)\n", *link, *depth, *timeout)

	if *depth > 0 {
		results, err := crawler.CrawlRecursive(*link, 0)
		if err != nil {
			log.Fatalf("Crawl failed: %v", err)
		}

		fmt.Printf("\nCrawled %d pages\n", len(results))

		if *llm && len(results) > 0 {
			avgRelevance := 0
			totalTopics := make(map[string]int)
			for _, result := range results {
				avgRelevance += result.Relevance
				for _, topic := range result.Topics {
					totalTopics[topic]++
				}
			}
			if len(results) > 0 {
				avgRelevance /= len(results)
			}

			fmt.Printf("LLM Insights Summary:\n")
			fmt.Printf("   Average Relevance: %d/10\n", avgRelevance)
			fmt.Printf("   Unique Topics Found: %d\n", len(totalTopics))

			if len(totalTopics) > 0 {
				fmt.Printf("   Most Common Topics: ")
				count := 0
				for topic, freq := range totalTopics {
					if count >= 3 {
						break
					}
					if count > 0 {
						fmt.Printf(", ")
					}
					fmt.Printf("%s (%d)", topic, freq)
					count++
				}
				fmt.Println()
			}
		}

		for _, result := range results {
			PrintResult(result, *verbose, *llm)
		}
	} else {
		result, err := crawler.Crawl(*link)
		if err != nil {
			log.Fatalf("Crawl failed: %v", err)
		}
		PrintResult(result, *verbose, *llm)
	}

	fmt.Printf("\nCrawl completed. Visited %d pages.\n", len(crawler.visited))
}
