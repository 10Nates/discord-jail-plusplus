package main

import (
	"database/sql"
	"fmt"
	"time"

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
jailed INTEGER NOT NULL
);

INSERT OR IGNORE INTO keyvalues(key, value) VALUES ('jailrole', '979912673703636992');` // default jail role ID

const newjaileduser string = `
INSERT INTO jailed(id, releasable, jailedtime, releasetime, reason, jailer, oldnick, oldpfpurl, oldroles, jailrole) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);
INSERT OR IGNORE INTO users(id, jailed) VALUES (?, 1);
UPDATE users SET jailed=1 WHERE id=?;`

const freeuser string = `
DELETE FROM jailed WHERE id=?;
UPDATE users SET jailed=0 WHERE id=?;`

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
	_, err := jaildb.Exec(`
	INSERT OR IGNORE INTO keyvalues(key, value) VALUES ('jailrole', ?);
	UPDATE keyvalues SET value=? WHERE key='jailrole';
	`, roleid, roleid)
	if err != nil {
		return err
	}

	return nil

}
