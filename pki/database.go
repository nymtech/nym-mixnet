/*
	Package pki implements basic functions for managing the pki
	represented as a SQL database.
*/

package pki

import (
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"database/sql"
	"errors"
)

/*
	OpenDatabase opens a connection with a specified database.
	OpenDatabase returns the database object and an error.
 */
func OpenDatabase(dataSourceName, dbDriver string) (*sqlx.DB, error) {

	var db *sqlx.DB
	db, err := sqlx.Connect(dbDriver, dataSourceName)

	if err != nil {
		return nil, err
	}

	return db, err
}

/*
	CreateTable creates a new table defined by a given name and specified
	column fields. CreateTable returns an error if a table could not be
	correctly created or when an SQL injection attacks was detected.

 */
func CreateTable(db *sqlx.DB, tableName string, params map[string]string) error{
	paramsAndTypes := make([]string, 0, len(params))

	for key := range params {
		paramsAndTypes = append(paramsAndTypes, key+" "+params[key])
	}

	paramsText := "idx INTEGER PRIMARY KEY, " + strings.Join(paramsAndTypes[:], ", ")

	if strings.ContainsAny(tableName, "'") || strings.ContainsAny(paramsText, "'"){
		return errors.New("detected ' character. Possible SQL injection")
	}
	if strings.ContainsAny(tableName, ";") || strings.ContainsAny(paramsText, ";"){
		return errors.New("detected ; character. Possible SQL injection")
	}

	query := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s ( %s )", tableName, paramsText)

	statement, err := db.Prepare(query)

	if err != nil{
		return err
	}
	_, err = statement.Exec()
	if err != nil{
		return err
	}

	return nil

}

/*
	InsertIntoTable allows to insert a new record into the specified table.
	The table name is checked for SQL injection attacks. The given input values
	are not explicitly checked, since the Exec build-in function should do this.
	The function returns an error if an SQL injection attack is detected or when
	insertion fails.
*/

func InsertIntoTable(db *sqlx.DB, tableName string, id, typ string, config []byte) error{
	if strings.ContainsAny(tableName, "'"){
		return errors.New("detected ' character. Possible SQL injection")
	}
	if strings.ContainsAny(tableName, ";"){
		return errors.New("detected ; character. Possible SQL injection")
	}

	query :="INSERT INTO " + tableName + " (Id, Typ, Config) VALUES (?, ?, ?)"

	stmt, err := db.Prepare(query)
	if err != nil{
		return err
	}
	_, err = stmt.Exec(id, typ, config)
	if err != nil{
		return err
	}

	return nil
}

/*
	QueryDatabase allows to query for records from a specified table, which
	Typ column satisfies a given condition. QueryDatabase checks for SQL injection
	in the tableName argument or condition argument. QueryDatabase returns a
	set of rows and an error.
*/

func QueryDatabase(db *sqlx.DB, tableName string, condition string) (*sqlx.Rows, error) {
	if strings.ContainsAny(tableName, "'") || strings.ContainsAny(condition, "'"){
		return nil, errors.New("detected ' character. Possible SQL injection")
	}
	if strings.ContainsAny(tableName, ";") || strings.ContainsAny(condition, ";"){
		return nil, errors.New("detected ; character. Possible SQL injection")
	}
	query := fmt.Sprintf("SELECT * FROM %s WHERE Typ = ?", tableName)
	rows, err := db.Queryx(query, condition)

	if err != nil {
		return nil, err
	}
	return rows, nil
}

/*
	rowExists checks whether a particular row, extracted using a given query, exists.
	rowExists is used only in the unit tests, hence doesn't have to contain the SQL injection attacks detection.
	If rowExists will become a public function, it should have SQL injection detection implemented.
 */
func rowExists(db *sqlx.DB, query string, args ...interface{}) (bool, error) {
	var exists bool
	query = fmt.Sprintf("SELECT exists (%s)", query)
	err := db.QueryRow(query, args...).Scan(&exists)
	if err != nil && err != sql.ErrNoRows {
		return false, err
	}
	return exists, nil
}