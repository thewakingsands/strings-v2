package server

import (
	"reflect"
	"slices"
	"testing"
	"xivstrings/pkg/store"
)

func TestParseLangs(t *testing.T) {
	cases := []struct {
		name    string
		raw     string
		want    []string
		wantErr string
	}{
		{
			name: "single language",
			raw:  "chs",
			want: []string{"chs"},
		},
		{
			name: "multiple languages keep the given order",
			raw:  "chs,en,ja",
			want: []string{"chs", "en", "ja"},
		},
		{
			name: "order is not normalised",
			raw:  "ja,en,chs",
			want: []string{"ja", "en", "chs"},
		},
		{
			name: "blank segments are dropped",
			raw:  "chs,,en",
			want: []string{"chs", "en"},
		},
		{
			name: "surrounding spaces are trimmed",
			raw:  " chs , en ",
			want: []string{"chs", "en"},
		},
		{
			name: "whitespace only segments are dropped",
			raw:  "chs,   ,en",
			want: []string{"chs", "en"},
		},
		{
			name: "duplicates are removed keeping first appearance",
			raw:  "chs,chs,en",
			want: []string{"chs", "en"},
		},
		{
			name: "duplicates do not change the preference order",
			raw:  "en,chs,en",
			want: []string{"en", "chs"},
		},
		{
			name: "every known language is accepted",
			raw:  "chs,tc,en,de,fr,ja,ko",
			want: []string{"chs", "tc", "en", "de", "fr", "ja", "ko"},
		},
		{
			name: "empty value yields no language and no error",
			raw:  "",
			want: []string{},
		},
		{
			name: "only separators yields no language and no error",
			raw:  ",,",
			want: []string{},
		},
		{
			name: "only whitespace yields no language and no error",
			raw:  "  ",
			want: []string{},
		},
		{
			name:    "unknown language is rejected",
			raw:     "xx",
			wantErr: "invalid lang: xx",
		},
		{
			name:    "unknown language among valid ones is rejected",
			raw:     "chs,xx,en",
			wantErr: "invalid lang: xx",
		},
		{
			name:    "language codes are case sensitive",
			raw:     "EN",
			wantErr: "invalid lang: EN",
		},
		{
			name:    "a field name is not a language",
			raw:     "sheet",
			wantErr: "invalid lang: sheet",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseLangs(tc.raw)

			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("parseLangs(%q) = %v, want error %q", tc.raw, got, tc.wantErr)
				}
				if err.Error() != tc.wantErr {
					t.Fatalf("parseLangs(%q) error = %q, want %q", tc.raw, err.Error(), tc.wantErr)
				}
				if got != nil {
					t.Errorf("parseLangs(%q) = %v on error, want nil", tc.raw, got)
				}
				return
			}

			if err != nil {
				t.Fatalf("parseLangs(%q) unexpected error: %v", tc.raw, err)
			}
			if !slices.Equal(got, tc.want) {
				t.Errorf("parseLangs(%q) = %v, want %v", tc.raw, got, tc.want)
			}
		})
	}
}

func TestParseSheetFilter(t *testing.T) {
	cases := []struct {
		name    string
		raw     string
		want    store.SheetFilter
		wantErr string
	}{
		{name: "empty value is no filter", raw: ""},
		{name: "only blank entries is no filter", raw: " , ,"},
		{
			name: "one exact name",
			raw:  "Item",
			want: store.SheetFilter{Include: []string{"Item"}},
		},
		{
			name: "patterns are trimmed and keep their order",
			raw:  " quest/* , cut_scene/* ",
			want: store.SheetFilter{Include: []string{"quest/*", "cut_scene/*"}},
		},
		{
			name: "only exclusions",
			raw:  "-Item,-Action",
			want: store.SheetFilter{Exclude: []string{"Item", "Action"}},
		},
		{
			name: "inclusion and exclusion",
			raw:  "quest/*,-quest/000/*",
			want: store.SheetFilter{Include: []string{"quest/*"}, Exclude: []string{"quest/000/*"}},
		},
		{
			name: "space after the minus is trimmed",
			raw:  "- Item",
			want: store.SheetFilter{Exclude: []string{"Item"}},
		},
		{name: "a lone minus is rejected", raw: "-", wantErr: "invalid sheet: -"},
		{name: "a blank exclusion is rejected", raw: "Item, - ", wantErr: "invalid sheet: -"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseSheetFilter(tc.raw)

			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("parseSheetFilter(%q) = %+v, want error %q", tc.raw, got, tc.wantErr)
				}
				if err.Error() != tc.wantErr {
					t.Fatalf("parseSheetFilter(%q) error = %q, want %q", tc.raw, err.Error(), tc.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("parseSheetFilter(%q) unexpected error: %v", tc.raw, err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("parseSheetFilter(%q) = %+v, want %+v", tc.raw, got, tc.want)
			}
		})
	}
}

func TestParseSearchMode(t *testing.T) {
	cases := []struct {
		raw     string
		want    bool
		wantErr string
	}{
		{raw: "", want: false},
		{raw: "simple", want: false},
		{raw: "advanced", want: true},
		{raw: " advanced ", want: true},
		{raw: "Advanced", wantErr: "invalid mode: Advanced"},
		{raw: "regex", wantErr: "invalid mode: regex"},
	}

	for _, tc := range cases {
		got, err := parseSearchMode(tc.raw)

		if tc.wantErr != "" {
			if err == nil || err.Error() != tc.wantErr {
				t.Errorf("parseSearchMode(%q) error = %v, want %q", tc.raw, err, tc.wantErr)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseSearchMode(%q) unexpected error: %v", tc.raw, err)
			continue
		}
		if got != tc.want {
			t.Errorf("parseSearchMode(%q) = %v, want %v", tc.raw, got, tc.want)
		}
	}
}
