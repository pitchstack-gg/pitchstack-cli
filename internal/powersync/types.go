package powersync

import (
	"context"
	"time"

	clientv1 "github.com/pitchstack-gg/pitchstack-go/client/v1"
)

const schemaVersion = "1"

type Credentials struct {
	Endpoint  string
	Token     string
	SyncEpoch string
}

type Connector interface {
	FetchCredentials(ctx context.Context) (*Credentials, error)
	UploadCrud(ctx context.Context, deviceID string, entries []clientv1.CrudEntry) (*clientv1.UploadCrudResponse, error)
}

type Operation struct {
	Bucket   string
	OpID     string
	Op       string
	Table    string
	ID       string
	Data     map[string]any
	Checksum string
}

type BucketPosition struct {
	Bucket string `json:"bucket"`
	OpID   string `json:"opId,omitempty"`
}

type Status struct {
	Path                string      `json:"path,omitempty"`
	SchemaVersion       string      `json:"schemaVersion,omitempty"`
	DeviceID            string      `json:"deviceId,omitempty"`
	SyncEpoch           string      `json:"syncEpoch,omitempty"`
	LastCheckpoint      string      `json:"lastCheckpoint,omitempty"`
	LastWriteCheckpoint string      `json:"lastWriteCheckpoint,omitempty"`
	LastSuccessfulSync  *time.Time  `json:"lastSuccessfulSync,omitempty"`
	Rows                int64       `json:"rows"`
	Buckets             int64       `json:"buckets"`
	PendingCrud         int64       `json:"pendingCrud"`
	FailedCrud          int64       `json:"failedCrud"`
	FailedEntries       []CrudError `json:"failedEntries,omitempty"`
}

type CrudError struct {
	OpID      int64  `json:"opId"`
	Op        string `json:"op"`
	Table     string `json:"table"`
	ID        string `json:"id"`
	Error     string `json:"error,omitempty"`
	UpdatedAt string `json:"updatedAt,omitempty"`
}

type CollectionCount struct {
	CollectionID    string `json:"collectionId"`
	Name            string `json:"name,omitempty"`
	ItemCount       int64  `json:"itemCount"`
	QuantityCount   int64  `json:"quantityCount"`
	UniqueCardCount int64  `json:"uniqueCardCount"`
}

type DeckListScope string

const (
	DeckListScopeAccessible DeckListScope = "accessible"
	DeckListScopeOwned      DeckListScope = "owned"
	DeckListScopeShared     DeckListScope = "shared"
)

type DeckListParams struct {
	Scope             DeckListScope
	ViewerUserID      string
	ViewerUserAliases []string
	Search            string
	Limit             int
}

type DeckSummary struct {
	ID                  string           `json:"id"`
	UserID              string           `json:"userId,omitempty"`
	Name                string           `json:"name,omitempty"`
	Author              string           `json:"author,omitempty"`
	HeroID              string           `json:"heroId,omitempty"`
	Format              string           `json:"format,omitempty"`
	Visibility          string           `json:"visibility,omitempty"`
	ActiveVersionID     string           `json:"activeVersionId,omitempty"`
	ActiveVersionName   string           `json:"activeVersionName,omitempty"`
	SelectedVersionID   string           `json:"selectedVersionId,omitempty"`
	SelectedVersionName string           `json:"selectedVersionName,omitempty"`
	DeckKind            string           `json:"deckKind,omitempty"`
	SourceKind          string           `json:"sourceKind,omitempty"`
	SourceReference     string           `json:"sourceReference,omitempty"`
	CreatedAt           string           `json:"createdAt,omitempty"`
	UpdatedAt           string           `json:"updatedAt,omitempty"`
	Ownership           string           `json:"ownership,omitempty"`
	VersionCount        int64            `json:"versionCount"`
	CardRowCount        int64            `json:"cardRowCount"`
	TotalQuantity       int64            `json:"totalQuantity"`
	ActiveVersionBoards []DeckBoardCount `json:"activeVersionBoards,omitempty"`
}

type DeckBoardCount struct {
	BoardType     string `json:"boardType"`
	CardRowCount  int64  `json:"cardRowCount"`
	TotalQuantity int64  `json:"totalQuantity"`
}

type DeckVersionSummary struct {
	ID        string `json:"id"`
	DeckID    string `json:"deckId,omitempty"`
	Name      string `json:"name,omitempty"`
	Notes     string `json:"notes,omitempty"`
	CreatedAt string `json:"createdAt,omitempty"`
	UpdatedAt string `json:"updatedAt,omitempty"`
}

type DeckCardLine struct {
	ID            string `json:"id"`
	DeckID        string `json:"deckId,omitempty"`
	DeckVersionID string `json:"deckVersionId,omitempty"`
	CardID        string `json:"cardId,omitempty"`
	BoardType     string `json:"boardType,omitempty"`
	Quantity      int64  `json:"quantity"`
	CreatedAt     string `json:"createdAt,omitempty"`
	UpdatedAt     string `json:"updatedAt,omitempty"`
}

type DeckDetails struct {
	Deck                DeckSummary          `json:"deck"`
	Versions            []DeckVersionSummary `json:"versions"`
	SelectedVersionID   string               `json:"selectedVersionId,omitempty"`
	SelectedVersionName string               `json:"selectedVersionName,omitempty"`
	HeroEquipment       []DeckCardLine       `json:"heroEquipment,omitempty"`
	Mainboard           []DeckCardLine       `json:"mainboard,omitempty"`
	Sideboard           []DeckCardLine       `json:"sideboard,omitempty"`
	Maybeboard          []DeckCardLine       `json:"maybeboard,omitempty"`
	Other               []DeckCardLine       `json:"other,omitempty"`
	HeroEquipmentCount  int64                `json:"heroEquipmentCount"`
	MainboardCount      int64                `json:"mainboardCount"`
	SideboardCount      int64                `json:"sideboardCount"`
	MaybeboardCount     int64                `json:"maybeboardCount"`
	OtherCount          int64                `json:"otherCount"`
	TotalQuantity       int64                `json:"totalQuantity"`
	CardRowCount        int64                `json:"cardRowCount"`
}

type InitResult struct {
	Path      string `json:"path,omitempty"`
	DeviceID  string `json:"deviceId,omitempty"`
	Endpoint  string `json:"endpoint,omitempty"`
	SyncEpoch string `json:"syncEpoch,omitempty"`
}
