package domain

import "time"

const (
	DigestSHA256       = "sha256"
	CustodyUserOwned   = "user-owned"
	ProviderFilesystem = "filesystem"

	RootIdentityStrong   = "strong"
	RootIdentityFallback = "fallback"
	RootIdentityWeak     = "weak"
	RootIdentityLegacy   = "legacy-unverified"
)

type ProviderRoot struct {
	SubmittedLocator   string `json:"submittedLocator"`
	OperationalLocator string `json:"operationalLocator"`
	FinalPathEvidence  string `json:"finalPathEvidence,omitempty"`
	PhysicalIdentity   string `json:"physicalIdentity,omitempty"`
	FallbackIdentity   string `json:"fallbackIdentity,omitempty"`
	IdentityConfidence string `json:"identityConfidence"`
	CatalogueOnly      bool   `json:"catalogueOnly"`
}

type Provider struct {
	ID          string `json:"id"`
	Kind        string `json:"kind"`
	DisplayName string `json:"displayName"`
	RootLocator string `json:"rootLocator"` // Compatibility alias for SubmittedLocator.
	ProviderRoot
	CreatedAt       time.Time  `json:"createdAt"`
	LastScanStarted *time.Time `json:"lastScanStarted,omitempty"`
	LastScanEnded   *time.Time `json:"lastScanEnded,omitempty"`
	ScanStatus      string     `json:"scanStatus"`
	ScanError       string     `json:"scanError,omitempty"`
}

type Content struct {
	ID         string `json:"id"`
	Algorithm  string `json:"algorithm"`
	DigestHex  string `json:"digest"`
	ByteLength int64  `json:"byteLength"`
}

type Appearance struct {
	ID             string    `json:"id"`
	ProviderID     string    `json:"providerId"`
	ContentID      string    `json:"contentId"`
	Locator        string    `json:"locator"`
	DisplayName    string    `json:"displayName"`
	NativeIdentity string    `json:"nativeIdentity,omitempty"`
	ByteLength     int64     `json:"byteLength"`
	ModifiedAt     time.Time `json:"modifiedAt"`
	Custody        string    `json:"custody"`
	Availability   string    `json:"availability"`
	Continuity     string    `json:"continuity"`
	ObservedAt     time.Time `json:"observedAt"`
}

type SearchResult struct {
	Content     Content      `json:"content"`
	Appearances []Appearance `json:"appearances"`
}

type CatalogueStats struct {
	Providers        int `json:"providers"`
	Contents         int `json:"contents"`
	AvailableFiles   int `json:"availableFiles"`
	UnavailableFiles int `json:"unavailableFiles"`
}

type ScanIssue struct {
	Locator string `json:"locator"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type FileObservation struct {
	Locator        string
	DisplayName    string
	DigestHex      string
	ByteLength     int64
	ModifiedAt     time.Time
	NativeIdentity string
}

type ProviderScan struct {
	StartedAt    time.Time
	EndedAt      time.Time
	Observations []FileObservation
	Issues       []ScanIssue
	Unstable     int
}

type ScanSummary struct {
	ProviderID string      `json:"providerId"`
	StartedAt  time.Time   `json:"startedAt"`
	EndedAt    time.Time   `json:"endedAt"`
	Observed   int         `json:"observed"`
	Unstable   int         `json:"unstable"`
	Issues     []ScanIssue `json:"issues"`
}
