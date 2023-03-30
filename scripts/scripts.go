package scripts

import _ "embed"

//go:embed schema.sql
var SchemaSQL string

//go:embed check_nitter_column.sql
var CheckNitterColumnSQL string

//go:embed create_nitter_column.sql
var CreateNitterColumnSQL string
