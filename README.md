# Migrator - pgx auto migrator

## Install

```bash
go get github.com/wassimbj/migrator
```

## Example

```go
import "github.com/wassimbj/migrator"

type User struct {
	Id          int    `db:"col:id;type:int"`
	Name        string `db:"col:name;type:varchar;size:20"`
	Email       string `db:"col:email;type:varchar;size:100"`
	Password    string `db:"col:password;type:varchar;size:20"`
	ConfirmCode string `db:"col:confirm_code;type:varchar;size:6"`
	CreatedAt   string `db:"col:created_at;type:timestamp"`
}

type Post struct {
	Id        int    `db:"col:id;type:int"`
	Title     string `db:"col:title;type:varchar;size:100"`
	Content   string `db:"col:content;type:varchar;size:1000"`
	UserId    string `db:"col:user_id;type:int"`
	CreatedAt string `db:"col:created_at;type:timestamp"`
}

func main() {

	conn, err := pgxpool.Connect(context.Background(), "postgres://root:1234@localhost:5432/testdb")
	if err != nil {
		fmt.Println("DB_CONN_ERROR ", err)
	}

	m := migrator.Init(conn)

	m.Migrate(&User{})
	m.Migrate(&Post{})

}

```

## TODO

- [ ] Test queries
- [ ] Add Auto Increment field
- [ ] Add indexes
- [ ] Add Primary keys
- [ ] Add Unique fields
- [ ] Add Foreign  Keys
