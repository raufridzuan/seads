package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
)

// URLScanSubmitter holds configurations for sending URL with unexpected domain to URLScan
type URLScanSubmitter struct {
	Token      string `yaml:"token"`
	ScanURL    string `yaml:"scanurl"`
	Tags       string `yaml:"tags"`
	Visibility string `yaml:"visibility"`
}

// URLScanSubmissionResponse represents the response from URLScan
type URLScanSubmissionResponse struct {
	Message    string  `json:"message"`
	UUID       string  `json:"uuid"`
	Result     string  `json:"result"`
	API        string  `json:"api"`
	Visibility string  `json:"visibility"`
	Options    Options `json:"options"`
	URL        string  `json:"url"`
	Country    string  `json:"country"`
}

type Options struct {
	UserAgent string `json:"useragent"`
}

// deduplicateURLs removes duplicate URLs from a list of AdResult objects
func deduplicateURLs(adsToScan []AdResult) []string {
	var uniqueAdLinks []string
	seenURLs := make(map[string]struct{})
	for _, ads := range adsToScan {
		if _, seen := seenURLs[ads.OriginalAdURL]; !seen {
			uniqueAdLinks = append(uniqueAdLinks, ads.OriginalAdURL)
			seenURLs[ads.OriginalAdURL] = struct{}{}
		}
	}
	return uniqueAdLinks
}

func (config *Config) URLScanSubmission(allAdResults []AdResult) {
	urlscanEndpoint := config.URLScanSubmitter.ScanURL
	token := config.URLScanSubmitter.Token
	visibility := config.URLScanSubmitter.Visibility
	tagsOriginal := config.URLScanSubmitter.Tags

	reNonAlphaNumeric := regexp.MustCompile(`[^a-zA-Z0-9]+`)
	rePaidForBy := regexp.MustCompile(`paid for by `)

	tagsOriginalList := strings.Split(tagsOriginal, ",")
	// Iterate allAdsResults
	for i := range allAdResults {
		// Process only having those having ExpectedDomains set to false
		// Check if for advertiser, ads location, engine name, and keyword
		tagList := tagsOriginalList
		ads := allAdResults[i]
		if ads.ExpectedDomains == true {
			continue
		}
		if ads.Engine != "" {
			engineTag := reNonAlphaNumeric.ReplaceAllString(ads.Engine, "_")
			engineTag = strings.ToLower(engineTag)
			engineTag = fmt.Sprintf("private.ads_engine_%s", engineTag)
			tagList = append(tagList, engineTag)
		}
		if ads.Query != "" {
			queryTag := reNonAlphaNumeric.ReplaceAllString(ads.Query, "_")
			queryTag = strings.ToLower(queryTag)
			queryTag = fmt.Sprintf("private.ads_keyword_%s", queryTag)
			if len(queryTag) > 30 {
				queryTag = queryTag[:30]
			}
			tagList = append(tagList, queryTag)
		}
		if ads.Advertiser != "" {
			adsTag := ads.Advertiser
			adsTag = strings.ToLower(adsTag)
			adsTag = rePaidForBy.ReplaceAllString(adsTag, "")
			adsTag = strings.Replace(adsTag, "paid for by", "", 1)
			adsTag = reNonAlphaNumeric.ReplaceAllString(adsTag, "_")
			adsTag = fmt.Sprintf("private.ads_name_%s", adsTag)
			if len(adsTag) > 30 {
				adsTag = adsTag[:30]
			}
			tagList = append(tagList, adsTag)
		}
		if ads.Location != "" {
			locTag := reNonAlphaNumeric.ReplaceAllString(ads.Location, "_")
			locTag = strings.ToLower(locTag)
			locTag = fmt.Sprintf("private.ads_loc_%s", locTag)
			if len(locTag) > 30 {
				locTag = locTag[:30]
			}
			tagList = append(tagList, locTag)
		}
		// if search engine
		if ads.IsSearchResults == true {
			tagList = append(tagList, "private.ads_search_result")
		}

		urlToScan := ads.OriginalAdURL

		urlscanResponse, err := SubmitURLScan(urlscanEndpoint, urlToScan, token, visibility, tagList)
		if err != nil {
			log.Fatalf("error submitting url scan: %v\n", err)
		}
		allAdResults[i].URLScan = urlscanResponse
	}

}

// SubmitURLScan submit single URL to the URLScan service for scanning
func SubmitURLScan(urlscanEndpoint string, urlToScan string, token string, visibility string, tagList []string) (URLScanSubmissionResponse, error) {
	log.Printf("\n*** URLScan Enabled ***\n")
	log.Printf("Endpoint URL: %v\n", urlscanEndpoint)
	log.Printf("Visibility: %v\n", visibility)
	log.Printf("Tags: %v\n", tagList)
	log.Printf("URL for submission: %s", urlToScan)

	data := map[string]interface{}{
		"url":        urlToScan,
		"visibility": visibility,
		"tags":       tagList,
	}

	// Convert data to JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Printf("Error marshaling JSON: %v\n", err)
		return URLScanSubmissionResponse{}, err
	}

	// Create a POST request object and set appropriate headers
	req, err := http.NewRequest("POST", urlscanEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Error creating request: %v", err)
		return URLScanSubmissionResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("API-Key", token)

	// Send the POST request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to send request: %v\n", err)
		return URLScanSubmissionResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("server returned non-200 status: %d\n", resp.StatusCode)
		return URLScanSubmissionResponse{}, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read response body: %v", err)
		return URLScanSubmissionResponse{}, err
	}

	var response URLScanSubmissionResponse
	if err := json.Unmarshal(body, &response); err != nil {
		log.Printf("Failed to parse JSON response: %v", err)
		return URLScanSubmissionResponse{}, err
	}

	log.Printf("*************************\n")
	log.Printf("Message: %s", response.Message)
	log.Printf("UUID: %s", response.UUID)
	log.Printf("Result URL: %s", response.Result)
	log.Printf("API URL: %s", response.API)
	log.Printf("Visibility: %s", response.Visibility)
	log.Printf("User Agent: %s", response.Options.UserAgent)
	log.Printf("Original URL: %s", response.URL)
	log.Printf("Country: %s", response.Country)
	log.Printf("\n*************************\n")

	return response, err

}
