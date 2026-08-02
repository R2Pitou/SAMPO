package domain

import "time"

const (
	DigestSHA256       = "sha256"
	CustodyUserOwned   = "user-owned"
	ProviderFilesystem = "filesystem"
)

type Provider struct {
	ID              string     `json:"id"`
	Kind            string     `json:"kind"`
	DisplayName     string     `json:"displayName"`
	RootLocator     string     `json:"rootLocator"`
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
