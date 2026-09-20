// Package events holds Wails event channel names.
// Convention: domain:eventName (colon, then lowerCamel). A typo in an event
// name does not fail the build, so names are constants on both sides.
package events

const (
	ImportProgress            = "import:progress"
	ImportCloseRequested      = "import:closeRequested"
	MaintenanceCloseRequested = "maintenance:closeRequested"
	DBUpdated                 = "db:updated"
	SearchIndexReady          = "search:indexReady"
	CoversProgress            = "covers:progress"
	StorageChanged            = "storage:changed"
)
