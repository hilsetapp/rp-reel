package evtlog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func write(t *testing.T, name, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoadJSONL(t *testing.T) {
	p := write(t, "log.jsonl", `{"ts":300,"type":"kill","player":"A","other":"B"}
{"ts":100,"type":"wreck","player":"C"}

{"ts":200,"type":"arrest","player":"D","other":"E"}
`)
	log, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(log.Events) != 3 {
		t.Fatalf("событий: %d, хочу 3", len(log.Events))
	}
	if log.Events[0].TS != 100 || log.Events[2].TS != 300 {
		t.Fatalf("не отсортировано по времени: %+v", log.Events)
	}
	if log.Skipped != 0 {
		t.Fatalf("пропущено строк: %d", log.Skipped)
	}
}

func TestLoadArray(t *testing.T) {
	p := write(t, "log.json", `[{"ts":10,"type":"kill","player":"A"},{"ts":20,"type":"rob","player":"B"}]`)
	log, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(log.Events) != 2 {
		t.Fatalf("событий: %d", len(log.Events))
	}
}

func TestBadLinesCounted(t *testing.T) {
	p := write(t, "log.jsonl", "не json\n{\"type\":\"kill\"}\n{\"ts\":5,\"type\":\"kill\",\"player\":\"A\"}\n")
	log, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(log.Events) != 1 {
		t.Fatalf("событий: %d, хочу 1", len(log.Events))
	}
	if log.Skipped != 2 {
		t.Fatalf("пропущено: %d, хочу 2", log.Skipped)
	}
	if len(log.Errors) != 2 {
		t.Fatalf("описаний ошибок: %d", len(log.Errors))
	}
}

func TestCaptions(t *testing.T) {
	cases := []struct {
		e    Event
		want string
	}{
		{Event{Type: Kill, Player: "Nick", Other: "Victim"}, "убийство: Nick → Victim"},
		{Event{Type: Kill, Player: "Nick"}, "убийство: Nick"},
		{Event{Type: Wreck, Player: "Sam"}, "разбивка: Sam"},
		{Event{Type: Arrest, Player: "Cop", Other: "Robber"}, "арест: Cop → Robber"},
		{Event{Type: Custom, Label: "ограбление банка"}, "ограбление банка"},
		{Event{Type: "bank_heist", Player: "Team"}, "bank_heist: Team"},
	}
	for _, c := range cases {
		if got := c.e.Caption(); got != c.want {
			t.Errorf("Caption() = %q, хочу %q", got, c.want)
		}
	}
	if !strings.Contains(TypeName(Wreck), "разбив") {
		t.Error("TypeName(wreck) должен быть по-русски")
	}
}
