package internal

import (
	"log"
	"time"

	gocql "github.com/gocql/gocql"
	gocqlx "github.com/scylladb/gocqlx"
	qb "github.com/scylladb/gocqlx/qb"
	table "github.com/scylladb/gocqlx/table"
)

type Rec = Record

// Создание кластера бд
func CreateCluster(consistency gocql.Consistency, keyspace string, hosts ...string) *gocql.ClusterConfig {
	retryPolicy := &gocql.ExponentialBackoffRetryPolicy{
		Min:        time.Second,
		Max:        10 * time.Second,
		NumRetries: 5,
	}
	cluster := gocql.NewCluster(hosts...)
	cluster.Keyspace = keyspace
	cluster.Timeout = 5 * time.Second
	cluster.RetryPolicy = retryPolicy
	cluster.Consistency = consistency
	cluster.PoolConfig.HostSelectionPolicy = gocql.TokenAwareHostPolicy(gocql.RoundRobinHostPolicy())
	return cluster
}

// инициализация кластера, сессии и подключение к бд
func ConnectDB() *gocql.Session {
	cluster := CreateCluster(gocql.Quorum, "isub", "127.0.0.1")
	session, err := gocql.NewSession(*cluster)
	if err != nil {
		log.Fatal("Ошибка подключения", err)
	}
	return session
}

var stmts = createStatements()

// создание выражений для работы с бд
func createStatements() *statements {
	//задание схемы таблицы
	m := table.Metadata{
		Name:    "users",
		Columns: []string{"id", "is_bot", "first_name", "last_name", "user_name", "language_code"},
		PartKey: []string{"id"},
	}
	tbl := table.New(m)

	//Методы работы с данными
	deleteStmt, deleteNames := tbl.Delete()
	insertStmt, insertNames := tbl.Insert()
	selectStmt, selectNames := qb.Select(m.Name).Columns(m.Columns...).ToCql()

	return &statements{
		del: query{
			stmt:  deleteStmt,
			names: deleteNames,
		},
		ins: query{
			stmt:  insertStmt,
			names: insertNames,
		},
		sel: query{
			stmt:  selectStmt,
			names: selectNames,
		},
	}
}

// Запрос на вставку
func InsertQuery(session *gocql.Session, user Record) {
	log.Println("Вставка " + user.ID + "......")

	r := Record{
		ID:           user.ID,
		IsBot:        user.IsBot,
		FirstName:    user.FirstName,
		LastName:     user.LastName,
		UserName:     user.UserName,
		LanguageCode: user.LanguageCode,
	}

	err := gocqlx.Query(session.Query(stmts.ins.stmt), stmts.ins.names).BindStruct(r).ExecRelease()
	if err != nil {
		log.Fatal("insert isub.users", err)
	}
}

// Запрос показа данных
func SelectQuery(session *gocql.Session, id string) bool {
	//Переменная для хранения считанной строки
	var record Rec

	scanner := gocqlx.Query(session.Query(stmts.sel.stmt), stmts.sel.names).Iter().Scanner()
	for scanner.Next() {
		err := scanner.Scan(&record.ID, &record.IsBot, &record.FirstName, &record.LastName, &record.UserName, &record.LanguageCode)
		if err != nil {
			log.Fatal(err)
		}

		//проверка на совпадение
		if record.ID == id {
			return true
		}
	}
	//Закрытие итератора
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
	return false
}
