package format

import "testing"

func TestFormatName(t *testing.T) {
	tables := []struct {
		input  string
		output string
	}{
		{"5.25", "5.25"},
		{" testing string", "TESTING_STRING"},
		{"mdroid core", "MDROID_CORE"},
		{"More tests       with spacing", "MORE_TESTS_WITH_SPACING"},
		{"true", "TRUE"},
		{" ", ""},
	}

	for _, table := range tables {
		got := FormatName(table.input)
		if got != table.output {
			t.Errorf("FormatName(%s) = %s; want %s", table.input, got, table.output)
		}
	}
}

func TestIsValidName(t *testing.T) {
	tables := []struct {
		input  string
		output bool
	}{
		{"5.25", true},
		{"TESTING_STRING", true},
		{"mdroid core", false},
		{"More tests       with spacing", false},
		{"true", false},
		{" ", false},
	}

	for _, table := range tables {
		got := IsValidName(table.input)
		if got != table.output {
			t.Errorf("IsValidName(%s) = %t; want %t", table.input, got, table.output)
		}
	}
}

func TestIsPositiveRequest(t *testing.T) {
	tables := []struct {
		input  string
		output bool
	}{
		{"ON", true},
		{"OFF", false},
		{"mdroid core", false},
	}

	for _, table := range tables {
		got, _ := IsPositiveRequest(table.input)
		if got != table.output {
			t.Errorf("IsPositiveRequest(%s) = %t; want %t", table.input, got, table.output)
		}
	}
}
