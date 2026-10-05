package database

import "strings"

// likeEscaper rende letterali i caratteri speciali di LIKE nel testo cercato,
// così "a_b" non corrisponde anche ad "axb". Il backslash è il carattere di
// escape predefinito di PostgreSQL.
var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// ContainsPattern trasforma il testo cercato in un pattern LIKE/ILIKE che
// trova i valori che lo contengono; con search vuoto trova tutto. Il
// pattern va sempre passato come parametro della query.
func ContainsPattern(search string) string {
	return "%" + likeEscaper.Replace(search) + "%"
}
