package visibility

import (
	"strings"
	"testing"
)

func TestListableWorkSQLCoversPersonalData(t *testing.T) {
	sql := strings.ToLower(ListableWorkSQL)
	for _, frag := range []string{"is_active = 1", "is_deleted = 0", "w.rating is not null", "w.comment", "w.want_to_read = 1"} {
		if !strings.Contains(sql, frag) {
			t.Fatalf("ListableWorkSQL missing %q", frag)
		}
	}
}

func TestExportableWorkSQLIsHistoryNotValues(t *testing.T) {
	sql := strings.ToLower(ExportableWorkSQL)
	for _, frag := range []string{"rating_updated_at is not null", "comment_updated_at is not null", "want_to_read_updated_at is not null"} {
		if !strings.Contains(sql, frag) {
			t.Fatalf("ExportableWorkSQL missing %q", frag)
		}
	}
	if strings.Contains(sql, "rating is not null") {
		t.Fatal("EXPORTABLE must not use value predicates")
	}
}

func TestListableTempSQLMatchesRule(t *testing.T) {
	joined := strings.ToLower(strings.Join(ListableTempSQL, "\n"))
	if !strings.Contains(joined, "primary key") {
		t.Fatal("listable temp table must be keyed by work_id")
	}
	if !strings.Contains(joined, "is_active = 1") || !strings.Contains(joined, "rating is not null") {
		t.Fatal("temp fill must include visible editions and personal data")
	}
	if !strings.Contains(joined, "want_to_read = 1") {
		t.Fatal("temp fill must include want_to_read")
	}
}
