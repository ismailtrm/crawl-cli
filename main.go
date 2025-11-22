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
	URL            string
	Title          string
	Links          []string
	Text           string
	Images         []string
	StatusCode     int
	ContentType    string
	Error          error
	Summary        string   `json:"summary,omitempty"`
	Topics         []string `json:"topics,omitempty"`
	Relevance      int      `json:"relevance,omitempty"`
	SuggestedLinks []string `json:"suggested_links,omitempty"`
}

// Crawler configuration
type Crawler struct {
	client          *http.Client
	maxDepth        int
	currentDepth    int
	visited         map[string]bool
	baseURL         *url.URL
	verbose         bool
	llmEnabled      bool
	geminiAPIKey    string
	crawlPurpose    string
	browserlessURL  string
	browserlessToken string
}

type GeminiRequest struct {
	Contents []GeminiContent `json:"contents"`
}

type GeminiContent struct {
	Parts []GeminiPart `json:"parts"`
}

type GeminiPart struct {
	Text string `json:"text"`
}

type GeminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

type LLMAnalysis struct {
	Summary        string   `json:"summary"`
	Topics         []string `json:"topics"`
	Relevance      int      `json:"relevance"`
	SuggestedLinks []string `json:"suggested_links"`
	ShouldCrawl    bool     `json:"should_crawl"`
	Priority       int      `json:"priority"`
}

func NewCrawler(timeout time.Duration, maxDepth int, verbose bool, llmEnabled bool, purpose string, insecure bool, browserlessURL string, browserlessToken string) *Crawler {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: insecure},
	}

	crawler := &Crawler{
		client: &http.Client{
			Timeout:   timeout,
			Transport: transport,
		},
		maxDepth:         maxDepth,
		visited:          make(map[string]bool),
		verbose:          verbose,
		llmEnabled:       llmEnabled,
		crawlPurpose:     purpose,
		browserlessURL:   browserlessURL,
		browserlessToken: browserlessToken,
	}

	if llmEnabled {
		crawler.geminiAPIKey = os.Getenv("GEMINI_API_KEY")
		if crawler.geminiAPIKey == "" {
			log.Fatal("GEMINI_API_KEY environment variable is required when using --llm flag")
		}
	}

	return crawler
}

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
		mode := "Standard HTTP"
		if c.browserlessURL != "" {
			mode = "Browserless (REST API)"
		}
		fmt.Printf("Crawling [%s]: %s\n", mode, targetURL)
	}

	var body []byte
	var contentType string
	var statusCode int

	// Choose between Browserless and Standard HTTP
	if c.browserlessURL != "" {
		body, err = c.fetchWithChrome(targetURL)
		if err != nil {
			return &CrawlResult{URL: targetURL, Error: err}, err
		}
		contentType = "text/html" // Browserless always returns rendered HTML
		statusCode = 200          // Assumption for successful rendering
	} else {
		resp, err := c.client.Get(targetURL)
		if err != nil {
			return &CrawlResult{URL: targetURL, Error: err}, err
		}
		defer resp.Body.Close()

		statusCode = resp.StatusCode
		contentType = resp.Header.Get("Content-Type")
		if !strings.Contains(contentType, "text/html") {
			return &CrawlResult{
				URL:         targetURL,
				StatusCode:  statusCode,
				ContentType: contentType,
			},
			nil
		}

		body, err = io.ReadAll(resp.Body)
		if err != nil {
			return &CrawlResult{URL: targetURL, Error: err}, err
		}
	}

	doc, err := html.Parse(strings.NewReader(string(body)))
	if err != nil {
		return &CrawlResult{URL: targetURL, Error: err}, err
	}

	result := &CrawlResult{
		URL:         targetURL,
		StatusCode:  statusCode,
		ContentType: contentType,
		Links:       []string{},
		Images:      []string{},
	}

	c.extractData(doc, result, parsedURL)

	if c.llmEnabled {
		err := c.enhanceWithLLM(result, string(body))
		if err != nil && c.verbose {
			fmt.Printf("LLM analysis failed: %v\n", err)
		}
	}

	return result, nil
}

// fetchWithChrome uses Browserless REST API (/content) to render the page
func (c *Crawler) fetchWithChrome(targetURL string) ([]byte, error) {
	// Clean up the base URL. If user provided "wss://", change to "https://"
	apiURL := c.browserlessURL
	apiURL = strings.Replace(apiURL, "wss://", "https://", 1)
	apiURL = strings.Replace(apiURL, "ws://", "http://", 1)
	
	// Ensure we hit the /content endpoint
	if !strings.HasSuffix(apiURL, "/content") {
		apiURL = strings.TrimSuffix(apiURL, "/")
		apiURL = apiURL + "/content"
	}

	// Add token if present
	if c.browserlessToken != "" {
		apiURL = fmt.Sprintf("%s?token=%s", apiURL, c.browserlessToken)
	}

	// Create JSON payload
	payload := map[string]interface{}{
		"url": targetURL,
		// Optional: Add wait conditions or other Browserless options here
		// "waitFor": 5000, // Wait 5 seconds (handled by server)
		// "rejectResourceTypes": []string{"image", "media"}, // Speed up by blocking images
	}
	
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	if c.verbose {
		fmt.Printf("DEBUG: Calling Browserless API: %s\n", apiURL)
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("browserless API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("browserless API error (status %d): %s", resp.StatusCode, string(body))
	}

	htmlContent, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read browserless response: %w", err)
	}

	if c.verbose {
		fmt.Printf("Browserless fetched %d bytes of HTML\n", len(htmlContent))
	}

	return htmlContent, nil
}

func (c *Crawler) enhanceWithLLM(result *CrawlResult, htmlContent string) error {
	prompt := c.buildAnalysisPrompt(result, htmlContent)
	analysis, err := c.callGemini(prompt)
	if err != nil {
		return err
	}
	llmResult, err := c.parseLLMResponse(analysis)
	if err != nil {
		return err
	}

	result.Summary = llmResult.Summary
	result.Topics = llmResult.Topics
	result.Relevance = llmResult.Relevance
	result.SuggestedLinks = llmResult.SuggestedLinks

	if c.verbose {
		fmt.Printf("[LLM Analysis] Relevance: %d/10 | Topics: %s\n", result.Relevance, strings.Join(result.Topics, ", "))
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
		purpose = "general web crawling"
	}

	return fmt.Sprintf(`Analyze this webpage. Purpose: %s. URL: %s. Title: %s. Content: %s.
	Return ONLY valid JSON:
	{
	  "summary": "Brief summary",
	  "topics": ["topic1", "topic2"],
	  "relevance": 8,
	  "suggested_links": ["url1"],
	  "should_crawl": true,
	  "priority": 7
	}`, purpose, result.URL, result.Title, result.Text[:min(2000, len(result.Text))])
}

func (c *Crawler) callGemini(prompt string) (string, error) {
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash:generateContent?key=%s", c.geminiAPIKey)
	request := GeminiRequest{Contents: []GeminiContent{{Parts: []GeminiPart{{Text: prompt}}}}}
	
	jsonData, _ := json.Marshal(request)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("API error: %d", resp.StatusCode)
	}

	var response GeminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", err
	}
	if len(response.Candidates) == 0 {
		return "", fmt.Errorf("empty response")
	}
	return response.Candidates[0].Content.Parts[0].Text, nil
}

func (c *Crawler) parseLLMResponse(response string) (*LLMAnalysis, error) {
	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimSuffix(response, "```")
	response = strings.TrimSpace(response)
	var analysis LLMAnalysis
	err := json.Unmarshal([]byte(response), &analysis)
	return &analysis, err
}

func (c *Crawler) extractData(n *html.Node, result *CrawlResult, baseURL *url.URL) {
	if n.Type == html.ElementNode {
		switch n.Data {
		case "title":
			if n.FirstChild != nil {
				result.Title = strings.TrimSpace(n.FirstChild.Data)
			}
		case "a":
			for _, attr := range n.Attr {
				if attr.Key == "href" {
					if link := c.resolveURL(attr.Val, baseURL); link != "" {
						result.Links = append(result.Links, link)
					}
				}
			}
		case "img":
			for _, attr := range n.Attr {
				if attr.Key == "src" {
					if imgURL := c.resolveURL(attr.Val, baseURL); imgURL != "" {
						result.Images = append(result.Images, imgURL)
					}
				}
			}
		case "p", "div", "h1", "h2", "article":
			if text := c.extractText(n); len(text) > 10 {
				result.Text += text + "\n"
			}
		}
	}
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		c.extractData(child, result, baseURL)
	}
}

func (c *Crawler) extractText(n *html.Node) string {
	if n.Type == html.TextNode {
		return strings.TrimSpace(n.Data)
	}
	var text string
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		text += " " + c.extractText(child)
	}
	return strings.TrimSpace(text)
}

func (c *Crawler) resolveURL(href string, base *url.URL) string {
	u, err := url.Parse(href)
	if err != nil || (u.Scheme != "" && u.Scheme != "http" && u.Scheme != "https") {
		return ""
	}
	return base.ResolveReference(u).String()
}

func (c *Crawler) CrawlRecursive(targetURL string, depth int) ([]*CrawlResult, error) {
	if depth > c.maxDepth {
		return nil, nil
	}
	result, err := c.Crawl(targetURL)
	if err != nil {
		return nil, err
	}
	
	results := []*CrawlResult{result}
	if depth < c.maxDepth {
		links := result.Links
		if c.llmEnabled && len(result.SuggestedLinks) > 0 {
			links = result.SuggestedLinks
		}
		for _, link := range links {
			if u, _ := url.Parse(link); u != nil && u.Host == c.baseURL.Host {
				sub, _ := c.CrawlRecursive(link, depth+1)
				results = append(results, sub...)
			}
		}
	}
	return results, nil
}

func PrintResult(result *CrawlResult, verbose bool, llmEnabled bool) {
	fmt.Printf("\n=== %s ===\n", result.URL)
	if result.Title != "" {
		fmt.Printf("Title: %s\n", result.Title)
	}
	if llmEnabled {
		fmt.Printf("Relevance: %d | Summary: %s\n", result.Relevance, result.Summary)
	}
	fmt.Printf("Links: %d | Images: %d\n", len(result.Links), len(result.Images))
}

func min(a, b int) int {
	if a < b { return a }
	return b
}

func main() {
	link := flag.String("link", "", "URL to crawl")
	depth := flag.Int("depth", 0, "Crawl depth")
	timeout := flag.Duration("timeout", 30*time.Second, "Timeout")
	verbose := flag.Bool("verbose", false, "Verbose output")
	llm := flag.Bool("llm", false, "Enable LLM")
	purpose := flag.String("purpose", "", "Crawl purpose")
	insecure := flag.Bool("insecure", false, "Skip TLS")
	browserless := flag.String("browserless", "", "Browserless URL (e.g., https://chrome.example.com)")
	token := flag.String("token", "", "Browserless API Token")
	scrape := flag.Bool("scrape", false, "Output raw HTML content to stdout and exit (no parsing/crawling)")

	flag.Parse()

	if *link == "" {
		fmt.Println("Usage: crawl-cli --link <URL> [--browserless <URL>] [--token <TOKEN>] [--scrape]")
		os.Exit(1)
	}

	// Check environment variable for token if flag is empty
	if *token == "" {
		*token = os.Getenv("BROWSERLESS_TOKEN")
	}

	// Check environment variable for browserless URL if flag is empty
	if *browserless == "" {
		*browserless = os.Getenv("BROWSERLESS_URL")
	}

	crawler := NewCrawler(*timeout, *depth, *verbose, *llm, *purpose, *insecure, *browserless, *token)

	// Scrape mode: Fetch and print raw HTML
	if *scrape {
		var body []byte
		var err error

		if *browserless != "" {
			body, err = crawler.fetchWithChrome(*link)
		} else {
			resp, reqErr := crawler.client.Get(*link)
			if reqErr != nil {
				err = reqErr
			} else {
				defer resp.Body.Close()
				body, err = io.ReadAll(resp.Body)
			}
		}

		if err != nil {
			log.Fatalf("Scrape failed: %v", err)
		}

		fmt.Print(string(body))
		return
	}
	
	if *depth > 0 {
		results, _ := crawler.CrawlRecursive(*link, 0)
		fmt.Printf("\nTotal pages crawled: %d\n", len(results))
	} else {
		result, _ := crawler.Crawl(*link)
		PrintResult(result, *verbose, *llm)
	}
}