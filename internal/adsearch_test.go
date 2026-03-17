package internal

import (
	"reflect"
	"strings"
	"testing"
)

func TestProcessAdResults(t *testing.T) {
	// Mock input data
	mockAdResults := []AdResult{
		{OriginalAdURL: "http://example.com", FinalDomainURL: "example.com", ExpectedDomains: true},
		{OriginalAdURL: "http://unexpected.com", FinalDomainURL: "unexpected.com", ExpectedDomains: false},
	}
	expectedDomainList := []string{"example.com"}

	// Mock output slices
	var allAdResults []AdResult

	// Enable necessary flags
	EnableNotifications = true
	PrintRedirectChain = false
	EnableURLScan = false

	// Mock config data
	config := Config{
		GlobalDomainExclusion: &GlobalDomainExclusion{
			GlobalDomainExclusionList: []string{},
		},
		Queries:          []SearchQuery{},
		URLScanSubmitter: nil, // Disable URLScan
	}

	// Call the function
	err := processAdResults(mockAdResults, expectedDomainList, &allAdResults, config)
	if err != nil {
		t.Fatalf("processAdResults returned an error: %v", err)
	}

	// Expected results
	expectedAllAdResults := mockAdResults

	// Assertions
	if !reflect.DeepEqual(allAdResults, expectedAllAdResults) {
		t.Errorf("allAdResults mismatch. Expected: %v, Got: %v", expectedAllAdResults, allAdResults)
	}
}

// TestGlobalDomainExclusionList ensures the global exclusion list is properly initialized and used.
func TestGlobalDomainExclusionList(t *testing.T) {
	// Mock input data
	// Here the domain inside GlobalDomainExclusionList should have ExpectedDomain flagged with true
	mockAdResults := []AdResult{
		{OriginalAdURL: "http://example.com", FinalDomainURL: "example.com", ExpectedDomains: true},
		{OriginalAdURL: "http://unexpected.com", FinalDomainURL: "unexpected.com", ExpectedDomains: false},
	}

	// Mock config data
	config := Config{
		GlobalDomainExclusion: &GlobalDomainExclusion{
			GlobalDomainExclusionList: []string{},
		},
		Queries:          []SearchQuery{},
		URLScanSubmitter: nil, // Disable URLScan
	}
	_ = config

	// Enable necessary flags
	EnableNotifications = true
	PrintRedirectChain = false
	EnableURLScan = false
	GlobalDomainExclusionList := []string{"example.com"}

	if GlobalDomainExclusionList == nil {
		t.Fatal("GlobalDomainExclusionList should not be nil. It has to be initialized.")
	}

	expectedGlobalDomainExclusionList := GlobalDomainExclusionList

	// Mock output slices
	var allAdResults []AdResult

	err := processAdResults(mockAdResults, expectedGlobalDomainExclusionList, &allAdResults, Config{})
	if err != nil {
		t.Errorf("processAdResults returned error: %v", err)
	}

	// Expected results
	expectedAllAdResults := mockAdResults

	// Assertions
	if !reflect.DeepEqual(allAdResults, expectedAllAdResults) {
		t.Errorf("allAdResults mismatch. Expected: %v, Got: %v", expectedAllAdResults, allAdResults)
	}
}

func TestIsInSelectedSearchEngineList(t *testing.T) {
	// Mock user defined list of SelectedEngine separated by comma
	SelectedEngine = "google, aol    ,syndicated, google1, typo-edsearch engine,yah oo, yahoo"

	// Split the string by comma
	parts := strings.Split(SelectedEngine, ",")
	var selectedEngineList []string
	for _, part := range parts {
		selectedEngineList = append(selectedEngineList, strings.TrimSpace(part))
	}

	// From available engine check for the Selected Engine
	testMockSelection := make(map[string]bool) // instantiate to all false
	for _, e := range searchEnginesFunctions {
		testMockSelection[e.EngineName] = false
	}

	for _, se := range selectedEngineList {
		if _, exists := testMockSelection[se]; exists {
			testMockSelection[se] = true
		}
	}

	if testMockSelection["google"] == false {
		t.Errorf("Search engine 'google' is in test map but not flagged as expected")
	}

	if testMockSelection["aol"] == false {
		t.Errorf("Search engine 'aol' is in test map but not flagged as expected")
	}

	if testMockSelection["syndicated"] == false {
		t.Errorf("Search engine 'syndicated' is in test map but not flagged as expected")
	}

	if testMockSelection["bing"] == true {
		t.Errorf("Search engine 'bing' is NOT test map but flagged as expected")
	}

	if _, exists := testMockSelection["google1"]; exists {
		t.Errorf("Search engine 'google1' is NOT a valid search engine but flagged as expected")
	}

	if _, exists := testMockSelection["typo-edsearch engine"]; exists {
		t.Errorf("Search engine 'typo-edsearch engine' is NOT a valid search engine but flagged as expected")
	}

	if _, exists := testMockSelection["yah oo"]; exists {
		t.Errorf("Search engine 'typo-edsearch engine' is NOT a valid search engine but flagged as expected")
	}

	if testMockSelection["yahoo"] == false {
		t.Errorf("Search engine 'aol' is in test map but not flagged as expected")
	}
}

func TestSearchEngineUnused(t *testing.T) {
	// Mock user defined list of SelectedEngine separated by comma
	SelectedEngine = ""

	// From available engine check if the Selected Engine
	testMockSelection := make(map[string]bool) // instantiate to all false

	// Keeping list of true engines
	trueEngineMap := make(map[string]bool)

	for _, engine := range searchEnginesFunctions {
		testMockSelection[engine.EngineName] = false
		trueEngineMap[engine.EngineName] = true
	}

	for _, engine := range searchEnginesFunctions {
		testMockSelection[engine.EngineName] = trueEngineMap[engine.EngineName]
	}

	for engineName, boolActive := range testMockSelection {
		if boolActive == false {
			t.Errorf("Search engine '%s' is in test map but not flagged as expected", engineName)
		}
	}
}
