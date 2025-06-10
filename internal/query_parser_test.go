package internal

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestScanner(t *testing.T) {
	tests := []struct {
		input string
		tok   token
		lit   string
	}{
		{"", tokenEOF, ""},

		{" ", tokenWS, " "},
		{"   ", tokenWS, "   "},
		{"\t", tokenWS, "\t"},
		{"\n", tokenWS, "\n"},

		{"(", tokenIllegal, "("},

		{"abcd", tokenIdent, "abcd"},
		{"{", tokenLeftBrace, "{"},
		{"}", tokenRightBrace, "}"},
		{",", tokenComma, ","},
		{":", tokenColon, ":"},
		{".", tokenDot, "."},
	}

	for i, tt := range tests {
		s := newQueryScanner(tt.input)
		tok, lit := s.scan()
		if tt.tok != tok {
			t.Errorf("%d. %q token mismatch: exp=%q got=%q <%q>", i, tt.input, tt.tok, tok, lit)
		} else if tt.lit != lit {
			t.Errorf("%d. %q literal mismatch: exp=%q got=%q", i, tt.input, tt.lit, lit)
		}
	}
}

// Ensure the parser can parse strings into Statement ASTs.
func TestParser_ParseStatement(t *testing.T) {
	tests := []struct {
		name  string
		input string
		query []*querySelector
		err   string
	}{
		{
			name:  "empty query",
			input: ``,
			query: nil,
		},
		{
			name:  "single field",
			input: `Person`,
			query: []*querySelector{
				{field: "Person"},
			},
		},
		{
			name:  "multiple fields",
			input: `Person Friends Questions`,
			query: []*querySelector{
				{field: "Person"},
				{field: "Friends"},
				{field: "Questions"},
			},
		},
		{
			name:  "dot is considered a field",
			input: `Person Friends . Questions`,
			query: []*querySelector{
				{field: "Person"},
				{field: "Friends"},
				{field: "."},
				{field: "Questions"},
			},
		},
		{
			name:  ": ignored",
			input: `:Person`,
			query: []*querySelector{
				{field: "Person"},
			},
		},
		{
			name:  "simple qualifier",
			input: `p:Person`,
			query: []*querySelector{
				{field: "Person", name: "p"},
			},
		},
		{
			name:  "empty props yields non-nil empty slice",
			input: `{}:Person`,
			query: []*querySelector{
				{field: "Person", props: []string{}},
			},
		},
		{
			name:  "props handled correctly",
			input: `{a, bb,    ccc}:Person`,
			query: []*querySelector{
				{field: "Person", props: []string{"a", "bb", "ccc"}},
			},
		},
		{
			name:  "qualifier and props",
			input: `a{b}:C`,
			query: []*querySelector{
				{field: "C", props: []string{"b"}, name: "a"},
			},
		},
		{
			name: "fails on duplicate qualifier",
		},
		{
			name:  "complex example",
			input: `p{name}:Person :FriendsWith . o{acquiredAt}:Owns pp:Pet`,
			query: []*querySelector{
				{field: "Person", props: []string{"name"}, name: "p"},
				{field: "FriendsWith"},
				{field: "."},
				{field: "Owns", props: []string{"acquiredAt"}, name: "o"},
				{field: "Pet", name: "pp"},
			},
		},

		{
			name:  "anything after a dot filed is unexpected",
			input: ".Person",
			err:   "expected next field or EOF after dot, got Person",
		},
		{
			name:  "anything other than ident and comma in props is unexpected",
			input: "{asdf,.}:.",
			err:   "expected field, comma or }, got .",
		},
		{
			name:  "unclosed brace is unexpected",
			input: "{:.",
			err:   "expected field, comma or }, got :",
		},
		{
			name:  "require colon between props and field",
			input: "{}.",
			err:   "expected colon after qualifier/props",
		},
		{
			name:  "require colon between props and field",
			input: "{}.",
			err:   "expected colon after qualifier/props",
		},
		{
			name:  "token after colon must be ident or dot",
			input: "{}::",
			err:   "unexpected token: :",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require := require.New(t)
			query, err := newQueryParser(test.input).parse()
			if err != nil {
				if test.err == "" {
					t.Fatalf("unexpected error: %s", errString(err))
				}
				require.ErrorContains(err, test.err)
			} else if test.err != "" {
				t.Fatalf("expected error: %s", test.err)
			}
			require.Equal(test.query, query)
		})
	}
}

func errString(err error) string {
	if err != nil {
		return err.Error()
	}
	return ""
}
