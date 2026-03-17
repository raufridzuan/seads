package internal

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"log"
	"math/rand"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode"
)

// initializeBrowser sets up the browser and returns the browser instance and the search results page
func initializeBrowser(query, searchEngineURL string) (*rod.Browser, *rod.Page, error) {
	chromePath, _ := launcher.LookPath()

	abs, err := filepath.Abs(BrowserProfile)
	if err != nil {
		return nil, nil, err
	}

	launcherURL := launcher.New().
		Bin(chromePath).
		UserDataDir(abs).
		Headless(false).
		Set("disable-blink-features", "AutomationControlled").
		Delete("enable-automation").
		Set("disable-features", "Translate").
		MustLaunch()
	browser := rod.New().ControlURL(launcherURL).MustConnect().MustIncognito()

	// Create blank page first
	page := browser.MustPage("about:blank") //.MustEmulate(Laptop)
	// page := stealth.MustPage(browser).MustEmulate(Laptop)

	// Test
	page.MustEvalOnNewDocument(`
		Object.defineProperty(navigator, 'webdriver', {
			get: () => undefined
		});
	`)

	page.MustNavigate(searchEngineURL + query)
	wait := page.MustWaitNavigation()
	wait()

	return browser, page, nil
}

// saveHTML saves the HTML content of the page to a file
func saveHTML(page *rod.Page, outputFilePrefix string, query string) {

	// Get the HTML content of the page
	htmlContent, err := page.HTML()
	if err != nil {
		log.Fatalf("failed to get HTML content: %v\n", err)
	}
	if Logger {
		log.Printf("Save search engine result is on\n")
	}
	fileHtmlPath := fmt.Sprintf("%s-%s-%s.html", outputFilePrefix, query, time.Now().Format("20060102-150405"))

	// Write the HTML content to a file
	err = os.WriteFile(filepath.Join(HtmlPath, fileHtmlPath), []byte(htmlContent), 0644)
	if err != nil {
		log.Fatalf("failed to save HTML to file: %v\n", err)
	} else {
		if Logger {
			log.Printf("Visited page saved to %s", fileHtmlPath)
		}
	}
}

// takeScreenshot saves a screenshot of the page to a file
func takeScreenshot(page *rod.Page, outputFilePrefix string, query string) {
	if Logger {
		log.Printf("Save screenshot is on\n")
	}
	filename := fmt.Sprintf("%s-%s-%s.png", outputFilePrefix, query, time.Now().Format("20060102-150405"))
	if Logger {
		log.Printf("Taking screenshot... ")
	}
	page.MustScreenshotFullPage(filepath.Join(ScreenshotPath, filename))
	if Logger {
		log.Printf("Screenshot saved at %s", filename)
	}
}

// getAdInfo retrieves the advertiser name and location, works only in google, syndicated and adsenseads
func getAdInfo(browser *rod.Browser, adDetail rod.Element) ([]string, error) {

	adInfoResult := []string{}

	// get ad info URL
	adInfoURL, err := adDetail.Attribute("href")
	if err != nil || adInfoURL == nil {
		return adInfoResult, fmt.Errorf("unable to find ad info URL: %v", err)
	}

	// navigate to ad info page
	adInfoPage := browser.MustPage()
	defer adInfoPage.Close()

	if Logger {
		log.Printf("Navigating to advertisement info page: %s\n", *adInfoURL)
	}

	err = adInfoPage.Navigate(*adInfoURL)
	if err != nil {
		return adInfoResult, fmt.Errorf("\nerror trying to open: %s -> %s\n", *adInfoURL, err)
	}
	adInfoPage.MustWaitLoad()

	// save advertiser name and location in advertisersInfo: [0] is name, [1] is location
	advertisersInfo, err := adInfoPage.ElementsX("//div[div[text()=\"Location\"]]/div[2]/text() | //div[div[text()=\"Location\"]]/preceding-sibling::div[1]/div[2]/text()")
	if err != nil || len(advertisersInfo) < 2 {
		return adInfoResult, fmt.Errorf("error trying to find ad details: %s -> %v\n", *adInfoURL, err)
	}

	// save advertiser name and location in advertisersInfo: [0] is name, [1] is location
	adInfoResult = append(adInfoResult, advertisersInfo[0].MustText(), advertisersInfo[1].MustText())

	return adInfoResult, nil
}

// extractAds extracts advertisement information from a search results page
func extractAds(browser *rod.Browser, page *rod.Page, userAgent, linkSelector, attrName, query, engine string, noRedirectionFlag bool) ([]AdResult, error) {
	// Find all ad elements on the page
	//// Set up concurrent processing
	//var (
	var adsFound []AdResult // Slice to store found ads
	//	mu       sync.Mutex     // Mutex to protect concurrent slice access
	//	wg       sync.WaitGroup // WaitGroup for goroutine synchronization
	//)
	adElements, err := page.Elements(linkSelector)
	if err != nil {
		return nil, fmt.Errorf("unable to find ad elements: %v", err)
	}

	if Logger {
		log.Printf("Total number of ad elements: %d\n", len(adElements))
	}

	if len(adElements) < 1 {
		return adsFound, nil
	}
	// Get additional ad details for Google-like engines (Google, Syndicated, AdsenseAds)
	var adDetails rod.Elements
	if isGoogleLikeEngine(engine) {
		adDetails, _ = page.Elements(adinfoSelector)
	}

	// Process each ad element concurrently
	// fmt.Println("adElements:", len(adElements))
	for i, adElement := range adElements {
		// Get the ad URL from the element
		adURL, err := adElement.Attribute(attrName)
		if err != nil || adURL == nil {
			// fmt.Println("unable to find ad URL")
			continue
		}
		//  fmt.Println("ad URL:", *adURL)

		//wg.Add(1)
		//go func(i int, adURL string) {
		//	defer wg.Done()
		//
		// Initialize basic ad information
		ad := AdResult{
			IsAds:         true,
			OriginalAdURL: *adURL,
			Query:         query,
			Time:          time.Now(),
			Engine:        engine,
		}

		// Extract advertiser info and location for Google-like engines
		if isGoogleLikeEngine(engine) && i < len(adDetails) {
			if Logger {
				log.Printf("isGoogleLikeEngine::getAdInfoSafe\n")
			}
			rand.Seed(time.Now().UnixNano())
			sleepSeconds := rand.Intn(3)
			log.Printf("Sleep for %d seconds before checking ads...\n", sleepSeconds)
			time.Sleep(time.Duration(sleepSeconds) * time.Second)
			ad.Advertiser, ad.Location = getAdInfoSafe(browser, adDetails[i])
		} else {

		}

		// Follow redirect chain if enabled
		if !noRedirectionFlag {
			ad.FinalRedirectURL, ad.FinalDomainURL = followAdRedirect(browser, *adURL, userAgent)
		}

		// Resolve ad URL for additional information
		ResolveAdUrl(*adURL, &ad)
		//
		//	// Safely append the ad to results
		//	mu.Lock()

		//	mu.Unlock()
		//}(i, *adURL)
		if engine == "google" && (ad.FinalRedirectURL == ad.OriginalAdURL) {
			// Then look for the other datapoint
			adURLAlt, err := adElement.Attribute("data-pcu")
			if err != nil || adURLAlt == nil {
				// pass
			} else {
				altURL := *adURLAlt

				// Get the first data
				parts := strings.Split(altURL, ",")
				if len(parts) >= 2 {
					altURL = strings.TrimSpace(parts[0])
					if altURL == "https://ad.doubleclick.net/" {
						altURL = strings.TrimSpace(parts[1])
					}
				}
				_, finalDomain := resolveAdURLByDomain(altURL)
				ad.FinalRedirectURL = altURL
				ad.FinalDomainURL = finalDomain
			}
		}
		adsFound = append(adsFound, ad)
	}

	// Wait for all goroutines to complete
	//wg.Wait()
	return adsFound, nil
}

// extractSearchResults extracts search-result information from a search results page
func extractSRs(browser *rod.Browser,
	page *rod.Page, userAgent, linkSelector, attrName, query,
	engine string, noRedirectionFlag bool) ([]AdResult, error) {
	var searchResultsFound []AdResult // Slice to store found ads
	srElements, err := page.Elements(linkSelector)

	if err != nil {
		return nil, fmt.Errorf("unable to find search result elements: %v", err)
	}

	if Logger {
		log.Printf("Total number of search result elements: %d\n", len(srElements))
	}

	if len(srElements) < 1 {
		return searchResultsFound, nil
	}

	// Process each ad element concurrently
	for _, srElement := range srElements {
		// Get the ad URL from the element
		srURL, err := srElement.Attribute(attrName)
		if err != nil || srURL == nil {
			continue
		}

		finalDomain, _ := extractDomain(*srURL)

		sr := AdResult{
			IsSearchResults:  true,
			Engine:           engine,
			Query:            query,
			OriginalAdURL:    *srURL,
			FinalDomainURL:   finalDomain,
			FinalRedirectURL: *srURL,
			Time:             time.Now(),
		}

		ResolveAdUrl(*srURL, &sr)

		if engine == "bing" && (sr.FinalRedirectURL == sr.OriginalAdURL) {
			// Then look for the other datapoint
			srURLAlt, err := srElement.Element("cite")
			if err != nil || srURLAlt == nil {
				fmt.Println("unable to find ad AltURL from cite")

			} else {
				srURLAlt := srURLAlt.MustText()
				re := regexp.MustCompile(`(https?://[^\s›]+)`)
				match := re.FindString(srURLAlt)
				_, finalDomain := resolveAdURLByDomain(match)
				sr.FinalRedirectURL = match
				sr.FinalDomainURL = finalDomain
				fmt.Printf("Final Redirect URL: %s\n", finalDomain)
			}
		}

		searchResultsFound = append(searchResultsFound, sr)
	}

	/// Validate search when no visitble destination onlink
	if engine == "google" {
		srAlts, err := page.Elements(`span[class="x2VHCd OSrXXb nMdasd ob9lvb"`)
		if err != nil {
			_ = fmt.Errorf("unable to find search result elements: %v", err)
		}
		fmt.Println("Total Search Results: ", len(srAlts))
		for _, srAlt := range srAlts {
			_, err := srAlt.Text()
			if err != nil {
				_ = fmt.Errorf("unable to find search result text: %v", err)
			}
			// Get the ad URL from the element
			_, err = srAlt.Attribute("data-dtld")
			if err != nil {
				_ = fmt.Errorf("unable to find search result attribute : data-dtld: %v", err)
			}
		}
	}

	return searchResultsFound, nil
}

// getAdInfoSafe safely extracts advertiser info and location
func getAdInfoSafe(browser *rod.Browser, adDetail *rod.Element) (advertiser, location string) {
	adInfo, err := getAdInfo(browser, *adDetail)
	if err == nil && len(adInfo) >= 2 {
		return adInfo[0], adInfo[1]
	}
	return "", ""
}

// followAdRedirect follows the ad URL and returns the final URL and domain
func followAdRedirect(browser *rod.Browser, adURL, userAgent string) (finalURL, finalDomain string) {
	adPage := browser.MustPage()
	defer adPage.Close()

	// Set custom user agent if provided
	if userAgent != "" {
		_ = adPage.SetUserAgent(&proto.NetworkSetUserAgentOverride{UserAgent: userAgent})
	}

	// Navigate to the ad URL and wait for completion
	wait := adPage.MustWaitNavigation()
	if err := adPage.Navigate(adURL); err == nil {
		wait()
		finalURL = adPage.MustInfo().URL
		finalDomain, _ = extractDomain(finalURL)
	}
	return
}

// ResolveAdUrl resolves the ad URL to its final destination and updates the AdResult
func ResolveAdUrl(adURL string, currentAd *AdResult) {
	// Skip resolution if already resolved
	if currentAd.FinalRedirectURL != "" {
		return
	}

	// First resolution attempt
	redirectURL, finalDomain := resolveAdURLByDomain(adURL)

	// Handle DoubleClick nested redirects
	if finalDomain == doubleclickdomain {
		redirectURL, finalDomain = resolveAdURLByDomain(redirectURL)
	}

	// Handle d.adx.io nested redirects
	if finalDomain == dadxio {
		redirectURL, finalDomain = resolveAdURLByDomain(redirectURL)
	}

	// Update the AdResult with final values
	currentAd.FinalRedirectURL = redirectURL
	currentAd.FinalDomainURL = finalDomain
}

// resolveAdURLByDomain handles URL resolution based on domain type
func resolveAdURLByDomain(adURL string) (string, string) {
	adDomain, err := extractDomain(adURL)
	if err != nil {
		if Logger {
			log.Printf("Error extracting domain from URL: %s", adURL)
		}
		return adURL, ""
	}

	// Resolve URL based on domain
	resolvers := map[string]func(string) (string, error){
		googledomain:      ResolveGoogleAdURL,
		adsenseadsdomain:  ResolveGoogleAdURL,
		syndicateddomain:  ResolveGoogleAdURL,
		bingdomain:        ResolveBingAdURL,
		ddgdomain:         ResolveDuckDuckGoAdURL,
		doubleclickdomain: ResolveDoubleClickAdURL,
		googleadsservices: ResolveGoogleAdURL,
		dadxio:            ResolveDadxioAdURL,
	}

	if resolver, exists := resolvers[adDomain]; exists {
		if resolvedURL, err := resolver(adURL); err == nil {
			finalDomain, _ := extractDomain(resolvedURL)
			return resolvedURL, finalDomain
		}
	}

	// Default case: return original URL and its domain
	return adURL, adDomain
}

// searchAdsWithEngine performs concurrent ad searches using a specific search engine
func searchAdsWithEngine(
	engineFunc func(string, string, string, bool) ([]AdResult, error),
	query, engineName string, userAgent string, noRedirection bool) ([]AdResult, error) {
	encodedQuery := url.QueryEscape(query)

	if Logger {
		log.Printf("Searching ads on %s\n", searchEngineURLs[engineName]+query)
	}

	// Collect ads using concurrent workers
	ads, err := runConcurrentSearch(engineFunc, encodedQuery, engineName, userAgent, noRedirection)
	if err != nil && len(ads) == 0 {

		return nil, fmt.Errorf("search failed for %s: %v", engineName, err)
	}

	// Process the collected ads
	return processSearchResults(ads, userAgent, noRedirection)
}

// runConcurrentSearch manages concurrent ad collection using worker pool
func runConcurrentSearch(
	engineFunc func(string, string, string, bool) ([]AdResult, error),
	query string, engineName string, userAgent string, noRedirection bool) ([]AdResult, error) {
	results := make(chan []AdResult, ConcurrencyLevel)
	errors := make(chan error, ConcurrencyLevel)

	// Sometimes the thread may encounter issues, thus creating timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var wg sync.WaitGroup

	// Launch workers
	for i := 0; i < ConcurrencyLevel; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			// Quick fix Recover from panics
			defer func() {
				if r := recover(); r != nil {
					log.Printf("\n\n******\n[Worker %d] Panic in runConcurrentSearch for %s\n*******\n\n", i, engineName)
				}
			}()

			resultChan := make(chan []AdResult, 1)
			errChan := make(chan error, 1)

			go func() {
				log.Printf("engineFunc::start worker '%d'", i)
				ads, err := engineFunc(query, userAgent, engineName, noRedirection)
				log.Printf("engineFunc::start end '%d'", i)
				if err != nil {
					errChan <- err
					return
				}
				resultChan <- ads
			}()

			// When timeout is triggered
			select {
			case <-ctx.Done():
				// Time out
				errors <- fmt.Errorf("Worker[%d] timeout waiting for search results for %s from %s: %v", i, query, engineName, ctx.Err())
				if Logger {
					log.Printf("Worker[%d] timeout waiting for search results for %s from %s: %v", i, query, engineName, ctx.Err())
				}
				return
			case err := <-errChan:
				errors <- err
			case ads := <-resultChan:
				results <- ads
			}
		}(i)
	}

	// Wait for completion and close channels
	go func() {
		wg.Wait()
		close(results)
		close(errors)
	}()

	// Collect results and handle errors
	var allAds []AdResult
	for ads := range results {
		allAds = append(allAds, ads...)
		log.Printf("Added %d results for %s\n", len(allAds), engineName)
	}

	// Calculate number of allAds
	if len(allAds) > 0 {
		log.Printf("Total allAds: %d\n", len(allAds))
	}

	// Check for errors
	for err := range errors {
		if err != nil {
			if len(allAds) > 0 {
				return allAds, err
			}
			log.Printf("Return with error searching ads: %v\n", err)
			return nil, err
		}
	}

	return allAds, nil
}

// processSearchResults handles post-search processing of ads and returns unique Ads
func processSearchResults(ads []AdResult, userAgent string, noRedirection bool) ([]AdResult, error) {
	// Remove duplicates
	uniqueAds, err := removeDuplicateAds(ads)
	if err != nil {
		return nil, fmt.Errorf("failed to remove duplicates: %v", err)
	}

	// Follow redirects if enabled
	if !noRedirection {
		for i := range uniqueAds {
			redirectChain, _ := findRedirectionChain(uniqueAds[i].OriginalAdURL, userAgent)
			uniqueAds[i].RedirectChain = redirectChain
		}
	}

	return uniqueAds, nil
}

func LoadArgumentExclusionList(filePath string) ([]string, error) {
	var exclusionList []string

	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.Contains(line, " ") {
			continue
		}

		// Check if the first character is not alphanumeric
		firstRune := []rune(line)[0]
		if !unicode.IsLetter(firstRune) && !unicode.IsDigit(firstRune) {
			continue
		}

		exclusionList = append(exclusionList, line)
	}
	log.Printf("Loaded %d exclusion list from file %s\n\n", len(exclusionList), filePath)
	return exclusionList, nil
}

// RunAdSearch returns the ads found in the search engines for the specified config
func RunAdSearch(config Config) ([]AdResult, error) {
	var allAdResults []AdResult

	// Get global domain exclusion list
	globalDomainExclusionList := config.GlobalDomainExclusion.GlobalDomainExclusionList

	// Load ExclusionList from Argument
	if len(ExclusionListFilePath) > 0 {
		argumentExclusionList, err := LoadArgumentExclusionList(ExclusionListFilePath)
		if err == nil {
			newGlobalList := append(globalDomainExclusionList, argumentExclusionList...)

			// Deduplicate using a map
			uniqueMap := make(map[string]bool)
			var deduped []string
			for _, item := range newGlobalList {
				if !uniqueMap[item] {
					uniqueMap[item] = true
					deduped = append(deduped, item)
				}
			}
			globalDomainExclusionList = deduped

		}
	}
	log.Printf("globalDomainExclusionList count: %d\n", len(globalDomainExclusionList))
	// If SelectedEngine is on
	if len(SelectedEngine) > 0 {
		log.Printf("Selected engine to use: %s", SelectedEngine)
		// Split the string by comma
		parts := strings.Split(SelectedEngine, ",")
		var cleaned []string
		for _, part := range parts {
			cleaned = append(cleaned, strings.TrimSpace(part))
		}
		// Create map for quick lookup
		engineMap := make(map[string]SearchEngineFunction)
		for _, se := range searchEnginesFunctions {
			engineMap[se.EngineName] = se
		}

		// Build ordered filtered list and report unknowns
		var filtered []SearchEngineFunction
		for _, name := range cleaned {
			if se, exists := engineMap[name]; exists {
				filtered = append(filtered, se)
			} else {
				log.Printf("\tWarning: '%s' not found in searchEnginesFunctions\n", name)
			}
		}

		if len(filtered) == 0 {
			log.Fatal("\tNo engines to search for")
		}
		searchEnginesFunctions = filtered

	}

	if DirectQuery != "" && len(DirectQuery) > 0 {
		log.Printf("DirectQuery SEARCH FOR: %s\n\n", DirectQuery)
		for _, engine := range searchEnginesFunctions {
			log.Printf("\n> Search Engine lookup using '%s' for keyword '%s'\n", engine.EngineName, DirectQuery)
			adResults, err := searchAdsWithEngine(engine.SearchFunction, DirectQuery, engine.EngineName, UserAgentString, NoRedirection)
			if err != nil {
				fmt.Printf("Error searching using %s: %v with adResult: %d\n", engine.EngineName, err, len(adResults))
				if len(adResults) == 0 {
					italic.Printf("  no ads and no search results found\n\n")
					continue
				} else {
					log.Printf("Continue process Ads")
				}
			}
			if len(adResults) == 0 {
				italic.Printf("  no ads found\n\n")
			} else {
				err := processAdResults(adResults, globalDomainExclusionList, &allAdResults, config)
				if err != nil {
					if len(adResults) != 0 {
						continue
					} else {
						log.Printf("Continue loop")
					}
				}
			}
		}
	} else {
		for _, searchQuery := range config.Queries {
			// Merge expected/exclusion individual expected domain with global domain lists
			expectedDomainList := mergeLists(globalDomainExclusionList, searchQuery.ExpectedDomains)
			log.Printf("\n* SEARCHING FOR: '%s'\n\n", searchQuery.SearchTerm)

			for _, engine := range searchEnginesFunctions {
				log.Printf("> Search Engine lookup using '%s' for keyword '%s'\n", engine.EngineName, searchQuery.SearchTerm)
				adResults, err := searchAdsWithEngine(engine.SearchFunction, searchQuery.SearchTerm, engine.EngineName, UserAgentString, NoRedirection)
				if err != nil {
					fmt.Printf("Error searching using %s: %v with adResult: %d\n", engine.EngineName, err, len(adResults))
					if len(adResults) == 0 {
						italic.Printf("  no ads and no search results found\n\n")
						continue
					} else {
						log.Printf("Continue process Ads and Search Results")
					}
				}
				if len(adResults) == 0 {
					italic.Printf("  no ads found\n\n")
				} else {
					err := processAdResults(adResults, expectedDomainList, &allAdResults, config)
					if err != nil {
						if len(adResults) != 0 {
							continue
						} else {
							log.Printf("Continue loop")
						}
					}
				}

			}
		}

	}

	return allAdResults, nil
}

// processAdResults processes the ad results and updates the respective lists
func processAdResults(adResults []AdResult, expectedDomainList []string, allAdResults *[]AdResult, config Config) error {
	// Iterate over each ad result
	for i := range adResults {
		adResult := adResults[i]

		if !IsExpectedDomain(adResult.FinalDomainURL, expectedDomainList) {
			if Logger {
				log.Printf("\nDomain '%s' not on expectedDomain\n", adResult.FinalDomainURL)
			}
			printDomainInfo(adResult, false)
			adResult.ExpectedDomains = false

			// Print the redirection chain if enabled
			if PrintRedirectChain {
				if err := printRedirectionChain(adResult.RedirectChain); err != nil {
					return fmt.Errorf("failed to print redirection chain: %w", err)
				}
			}
		} else {
			// add is in the expected domain list
			printDomainInfo(adResult, true)
			adResult.ExpectedDomains = true
		}
		// Append the ad result to the allAdResults list
		*allAdResults = append(*allAdResults, adResult)
	}
	return nil
}

// processAdResults processes the ad results and updates the respective lists
func processSRResults(srResults []AdResult, expectedDomainList []string, allSRResults *[]AdResult, config Config) error {
	// Iterate over each ad result
	for i := range srResults {
		srResult := srResults[i]

		if !IsExpectedDomain(srResult.FinalDomainURL, expectedDomainList) {
			if Logger {
				log.Printf("\nDomain '%s' not on expectedDomain\n", srResult.FinalDomainURL)
			}
			srResult.ExpectedDomains = false

		} else {
			// add is in the expected domain list
			srResult.ExpectedDomains = true
		}
		// Append the ad result to the allAdResults list
		*allSRResults = append(*allSRResults, srResult)
	}
	return nil
}

var ErrPageEmpty = errors.New("Page is empty")

func isPageEmpty(page *rod.Page) (bool, error) {
	el, err := page.Element("body")
	if err != nil {
		return true, err
	}
	text, err := el.Text()
	if err != nil {
		return true, err
	}

	if len(strings.TrimSpace(text)) == 0 {
		return true, ErrPageEmpty
	}

	return false, nil

}

var ErrChallengePageDetected = errors.New("Challenge or bot-detection page detected")

func isChallengePage(page *rod.Page) (bool, error) {
	el, err := page.Element("body")
	if err != nil {
		return false, err
	}
	text, err := el.Text()
	if err != nil {
		return false, err
	}
	text = strings.ToLower(text)
	//fmt.Print(text)

	indicators := []string{
		"unusual traffic from your computer",
		"verify you are a human",
		"please solve the challenge below to continue",
	}

	for _, s := range indicators {
		if strings.Contains(text, s) {
			return true, ErrChallengePageDetected
		}
	}
	return false, nil
}
