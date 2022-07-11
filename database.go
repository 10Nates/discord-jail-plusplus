package main

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/andersfylling/snowflake/v5"
	_ "github.com/mattn/go-sqlite3"
)

const file string = "jail.db"

type JailedUser struct {
	id          uint64 // discord ID
	releasable  bool   // whether or not to release them
	jailedTime  time.Time
	releaseTime time.Time
	reason      string
	jailer      uint64 // person who jailed them
	oldnick     string // nickname at time of jail
	oldpfpurl   string // profile picture at time of jail
	oldroles    string // role IDs separated by spaces
	jailrole    string // role given to them when they were jailed
}

type User struct {
	id     uint64 // discord ID
	jailed bool
	marks  string // mark IDs separated by spaces
}

type Mark struct {
	id   uint64 // role ID for mark & mark ID
	name string // refer to mark
}

const create string = `
CREATE TABLE IF NOT EXISTS jailed (
id INTEGER NOT NULL PRIMARY KEY,
releasable INTEGER NOT NULL,
jailedtime DATETIME NOT NULL,
releasetime DATETIME NOT NULL,
reason TEXT,
jailer INTEGER NOT NULL,
oldnick TEXT NOT NULL,
oldpfpurl TEXT,
oldroles TEXT,
jailrole TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS keyvalues (
key TEXT NOT NULL PRIMARY KEY,
value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS users (
id INTEGER NOT NULL PRIMARY KEY,
jailed INTEGER NOT NULL,
marks TEXT
);

CREATE TABLE IF NOT EXISTS marks (
id INTEGER NOT NULL PRIMARY KEY,
name TEXT NOT NULL
);

INSERT OR IGNORE INTO keyvalues(key, value) VALUES ('jailrole', '979912673703636992');` // default jail role ID

const newjaileduser string = `
INSERT INTO jailed(id, releasable, jailedtime, releasetime, reason, jailer, oldnick, oldpfpurl, oldroles, jailrole) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
INSERT OR IGNORE INTO users(id, jailed) VALUES (?, 1);
UPDATE users SET jailed=1 WHERE id=?;`

const freeuser string = `
DELETE FROM jailed WHERE id=?;
UPDATE users SET jailed=0 WHERE id=?;`

const setjailrole string = `
INSERT OR IGNORE INTO keyvalues(key, value) VALUES ('jailrole', ?);
UPDATE keyvalues SET value=? WHERE key='jailrole';`

const newmark string = `
INSERT INTO marks(id, name) VALUES (?, ?);`

const delmark string = `
DELETE FROM marks WHERE id=?`

var jaildb *sql.DB

func InitDB() {
	db, err := sql.Open("sqlite3", file)
	if err != nil {
		panic(err)
	}
	if _, err := db.Exec(create); err != nil {
		panic(err)
	}
	jaildb = db

	err = jaildb.Ping()
	if err != nil {
		panic(err)
	}

	fmt.Println("Database initialized")
}

// jail stuff

func QueryJail(query string, args ...interface{}) ([]*JailedUser, error) {
	rows, err := jaildb.Query(query, args...)
	if err != nil {
		return nil, err
	}

	data := []*JailedUser{}

	for rows.Next() {
		i := JailedUser{}
		err := rows.Scan(&i.id, &i.releasable, &i.jailedTime, &i.releaseTime, &i.reason, &i.jailer, &i.oldnick, &i.oldpfpurl, &i.oldroles, &i.jailrole)
		if err != nil {
			return nil, err
		}
		data = append(data, &i)
	}

	return data, nil
}

func FetchJailedUser(id uint64) (*JailedUser, error) {
	users, err := QueryJail("SELECT id, releasable, jailedtime, releasetime, reason, jailer, oldnick, oldpfpurl, oldroles, jailrole FROM jailed WHERE id=?", id)
	if err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return nil, fmt.Errorf("could not find user in database, please try again")
	} else if len(users) > 1 {
		return nil, fmt.Errorf("more than one user found with same id, something terrible happened")
	}

	return users[0], nil
}

//whether or not they are to be released
func FetchAllJailedUsers(releasableOnly bool) ([]*JailedUser, error) {
	var users []*JailedUser
	var err error
	if releasableOnly {
		users, err = QueryJail("SELECT id, releasable, jailedtime, releasetime, reason, jailer, oldnick, oldpfpurl, oldroles, jailrole FROM jailed WHERE releasable=1") // 1 == true
	} else {
		users, err = QueryJail("SELECT id, releasable, jailedtime, releasetime, reason, jailer, oldnick, oldpfpurl, oldroles, jailrole FROM jailed")
	}
	if err != nil {
		return nil, err
	}

	return users, nil
}

func JailNewUser(i *JailedUser) (*sql.Result, error) {
	res, err := jaildb.Exec(newjaileduser, i.id, i.releasable, i.jailedTime, i.releaseTime, i.reason, i.jailer, i.oldnick, i.oldpfpurl, i.oldroles, i.jailrole, i.id, i.id)
	return &res, err
}

func RemoveJailedUser(id uint64) (*sql.Result, error) {
	res, err := jaildb.Exec(freeuser, id, id)
	if err != nil {
		return nil, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}
	if affected == 1 {
		return &res, nil
	} else {
		return &res, fmt.Errorf("more than one user deleted")
	}
}

func GetJailRole() (string, error) {
	row := jaildb.QueryRow("SELECT value FROM keyvalues WHERE key='jailrole'")
	var roleid string
	err := row.Scan(&roleid)
	if err != nil {
		return "", err
	}

	return roleid, nil
}

func SetJailRole(roleid string) error {
	_, err := jaildb.Exec(setjailrole, roleid, roleid)
	if err != nil {
		return err
	}

	return nil
}

// marking stuff

func QueryMarks(query string, args ...interface{}) ([]*Mark, error) {
	rows, err := jaildb.Query(query, args...)
	if err != nil {
		return nil, err
	}

	data := []*Mark{}

	for rows.Next() {
		i := Mark{}
		err := rows.Scan(&i.id, &i.name)
		if err != nil {
			return nil, err
		}
		data = append(data, &i)
	}

	return data, nil
}

func FetchMarkByName(name string) (*Mark, error) {
	marks, err := QueryMarks("SELECT id, name FROM jailed WHERE name=?", name)
	if err != nil {
		return nil, err
	}
	if len(marks) == 0 {
		return nil, fmt.Errorf("could not find mark in database, please try again")
	} else if len(marks) > 1 {
		return nil, fmt.Errorf("more than one mark found with same name, something terrible happened")
	}

	return marks[0], nil
}

func FetchMarkByID(id uint64) (*Mark, error) {
	marks, err := QueryMarks("SELECT id, name FROM jailed WHERE id=?", id)
	if err != nil {
		return nil, err
	}
	if len(marks) == 0 {
		return nil, fmt.Errorf("could not find mark in database, please try again")
	} else if len(marks) > 1 {
		return nil, fmt.Errorf("more than one mark found with same id, something terrible happened")
	}

	return marks[0], nil
}

func FetchUserByID(id uint64) (*User, error) {
	rows, err := jaildb.Query("SELECT id, jailed, marks FROM users WHERE id=?", id)
	if err != nil {
		return nil, err
	}

	users := []*User{}

	for rows.Next() {
		i := User{}
		err := rows.Scan(&i.id, &i.jailed, &i.marks)
		if err != nil {
			return nil, err
		}
		users = append(users, &i)
	}

	if len(users) == 0 {
		return nil, fmt.Errorf("could not find user in database, please try again")
	} else if len(users) > 1 {
		return nil, fmt.Errorf("more than one user found with same id, something terrible happened")
	}

	return users[0], nil
}

func FetchUserMarks(id uint64) ([]*Mark, error) {
	user, err := FetchUserByID(id)
	if err != nil {
		return nil, err
	}

	marks := []*Mark{}

	markliststring := user.marks
	marklist := strings.Split(markliststring, " ")
	for i := 0; i < len(marklist); i++ {
		mark, err := FetchMarkByID(uint64(snowflake.ParseSnowflakeString(marklist[i])))
		if err != nil {
			return nil, err
		}
		marks = append(marks, mark)
	}

	return marks, nil
}

func AddMark(id uint64, name string) (*sql.Result, error) {
	res, err := jaildb.Exec(newmark, id, name)
	return &res, err
}

func DeleteMark(id uint64) (*sql.Result, error) {
	res, err := jaildb.Exec(delmark, id)
	return &res, err
}
