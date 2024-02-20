package internal

type UserExistence struct {
	ID int64 `json:"id"`
}

type query struct {
	stmt  string
	names []string
}

type statements struct {
	del query
	ins query
	sel query
}

type Record struct {
	ID           string `db:"id"`
	IsBot        string `db:"is_bot"`
	FirstName    string `db:"first_name"`
	LastName     string `db:"last_name"`
	UserName     string `db:"user_name"`
	LanguageCode string `db:"language_code"`
}
