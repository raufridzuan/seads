package internal

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
)

func init() {

	// Extract EngineName
	var engineNames []string
	for _, se := range searchEnginesFunctions {
		engineNames = append(engineNames, se.EngineName)
	}
	var engineNamesList = strings.Join(engineNames, ",")
	engineNamesListUsage := fmt.Sprintf("Selected engine to use: %s", engineNamesList)

	flag.StringVar(&ConfigFilePath, "config", ConfigFilePath, "path to config file (default \"config.yaml\")")
	flag.IntVar(&ConcurrencyLevel, "concurrency", ConcurrencyLevel, "number of concurrent headless browsers (default 4)")
	flag.StringVar(&ScreenshotPath, "screenshot", ScreenshotPath, "path to store screenshots (if empty, the screenshot feature will be disabled)")
	flag.BoolVar(&PrintCleanLinks, "cleanlinks", PrintCleanLinks, "print clear links in output (links will remain defanged in notifications)")
	flag.BoolVar(&EnableNotifications, "notify", EnableNotifications, "notify if unexpected domains are found (requires notifications fields in config.yaml)")
	flag.BoolVar(&PrintRedirectChain, "printredirect", PrintRedirectChain, "print redirection chain for ad links found")
	flag.StringVar(&UserAgentString, "ua", UserAgentString, "User-Agent string to be used to click on ads")
	flag.StringVar(&OutputFilePath, "out", OutputFilePath, "path of JSON file containing links of gathered ads")
	flag.BoolVar(&EnableURLScan, "urlscan", EnableURLScan, "submit url to urlscan.io for analysis")
	flag.BoolVar(&NoRedirection, "noredirect", NoRedirection, "do not follow redirection; if \"urlscan\" is enabled, submit advertisement link to resolve by URLScan instead")
	flag.StringVar(&HtmlPath, "html", HtmlPath, "path to store search engine result html page (if empty, the htmlPath feature will be disabled)")
	flag.BoolVar(&Logger, "log", Logger, "enable detailed logging, VERY VERBOSE!")
	flag.StringVar(&DirectQuery, "directquery", DirectQuery, "Direct query from command line and not using queries on config file")
	flag.StringVar(&SelectedEngine, "selectedengine", SelectedEngine, engineNamesListUsage)
	flag.StringVar(&ExclusionListFilePath, "exclusionListFilePath", ExclusionListFilePath, "File with list of domains for global exclusion")
	flag.StringVar(&BrowserProfile, "browserprofile", BrowserProfile, "Profile to use for browser profile")
	log.SetFlags(0)
}

// ShowHelp displays the help message and usage information
func ShowHelp() {
	fmt.Printf("Usage:\n\n")
	fmt.Printf("utils -config <config.yaml> [options]\n\n")
	flag.PrintDefaults()
	fmt.Println()

	os.Exit(1)
}

// PrintFlags prints the current values of the command-line arguments
func PrintFlags() {
	log.Println("Configuration Flags:")
	log.Printf("  ConfigFilePath: %s\n", ConfigFilePath)
	log.Printf("  ConcurrencyLevel: %d\n", ConcurrencyLevel)
	log.Printf("  ScreenshotPath: %s\n", ScreenshotPath)
	log.Printf("  PrintCleanLinks: %t\n", PrintCleanLinks)
	log.Printf("  EnableNotifications: %t\n", EnableNotifications)
	log.Printf("  PrintRedirectChain: %t\n", PrintRedirectChain)
	log.Printf("  UserAgentString: %s\n", UserAgentString)
	log.Printf("  EnableURLScan: %t\n", EnableURLScan)
	log.Printf("  OutputFilePath: %s\n", OutputFilePath)
	log.Printf("  NoRedirection: %t\n", NoRedirection)
	log.Printf("  HtmlPath: %s\n", HtmlPath)
	log.Printf("  Logger: %t\n", Logger)
	log.Printf("  DirectQuery: %s\n", DirectQuery)
	log.Printf("  SelectedEngine: %s\n", SelectedEngine)
	log.Printf("  ExclusionList: %s\n\n", ExclusionListFilePath)
}
