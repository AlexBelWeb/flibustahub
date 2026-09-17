package visibility

import (
	"strings"
	"testing"
)

func TestListableWorkSQLCoversPersonalData(t *testing.T) {
	sql := strings.ToLower(ListableWorkSQL)
	for _, frag := range []string{"is_active = 1", "is_deleted = 0", "w.rating is not null", "w.comment"} {
		if !strings.Contains(sql, frag) {
			t.Fatalf("ListableWorkSQL missing %q", frag)
		}
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
}
