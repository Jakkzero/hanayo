package v1

import (
	"crypto/md5"
	"database/sql"
	"fmt"

	"git.zxq.co/ripple/rippleapi/common"
	"git.zxq.co/ripple/schiavolib"
	"golang.org/x/crypto/bcrypt"
)

type tokenNewInData struct {
	// either username or userid must be given in the request.
	// if none is given, the request is trashed.
	Username    string `json:"username"`
	UserID      int    `json:"id"`
	Password    string `json:"password"`
	Privileges  int    `json:"privileges"`
	Description string `json:"description"`
}

type tokenNewResponse struct {
	common.ResponseBase
	Username   string `json:"username"`
	ID         int    `json:"id"`
	Privileges int    `json:"privileges"`
	Token      string `json:"token,omitempty"`
	Banned     bool   `json:"banned"`
}

// TokenNewPOST is the handler for POST /token/new.
func TokenNewPOST(md common.MethodData) common.CodeMessager {
	var r tokenNewResponse
	data := tokenNewInData{}
	err := md.RequestData.Unmarshal(&data)
	if err != nil {
		return ErrBadJSON
	}

	var miss []string
	if data.Username == "" && data.UserID == 0 {
		miss = append(miss, "username|id")
	}
	if data.Password == "" {
		miss = append(miss, "password")
	}
	if len(miss) != 0 {
		return ErrMissingField(miss...)
	}

	var q *sql.Row
	const base = "SELECT id, username, rank, password_md5, password_version, allowed FROM users "
	if data.UserID != 0 {
		q = md.DB.QueryRow(base+"WHERE id = ? LIMIT 1", data.UserID)
	} else {
		q = md.DB.QueryRow(base+"WHERE username = ? LIMIT 1", data.Username)
	}

	var (
		rank      int
		pw        string
		pwVersion int
		allowed   int
	)

	err = q.Scan(&r.ID, &r.Username, &rank, &pw, &pwVersion, &allowed)
	switch {
	case err == sql.ErrNoRows:
		return common.SimpleResponse(404, "No user with that username/id was found.")
	case err != nil:
		md.Err(err)
		return Err500
	}

	if nFailedAttempts(r.ID) > 20 {
		return common.SimpleResponse(429, "You've made too many login attempts. Try again later.")
	}

	if pwVersion == 1 {
		return common.SimpleResponse(418, "That user still has a password in version 1. Unfortunately, in order for the API to check for the password to be OK, the user has to first log in through the website.")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(pw), []byte(fmt.Sprintf("%x", md5.Sum([]byte(data.Password))))); err != nil {
		if err == bcrypt.ErrMismatchedHashAndPassword {
			go addFailedAttempt(r.ID)
			return common.SimpleResponse(403, "That password doesn't match!")
		}
		md.Err(err)
		return Err500
	}
	if allowed == 0 {
		r.Code = 200
		r.Message = "That user is banned."
		r.Banned = true
		return r
	}
	r.Privileges = int(common.Privileges(data.Privileges).CanOnly(rank))

	var (
		tokenStr string
		tokenMD5 string
	)
	for {
		tokenStr = common.RandomString(32)
		tokenMD5 = fmt.Sprintf("%x", md5.Sum([]byte(tokenStr)))
		r.Token = tokenStr
		id := 0

		err := md.DB.QueryRow("SELECT id FROM tokens WHERE token=? LIMIT 1", tokenMD5).Scan(&id)
		if err == sql.ErrNoRows {
			break
		}
		if err != nil {
			md.Err(err)
			return Err500
		}
	}
	_, err = md.DB.Exec("INSERT INTO tokens(user, privileges, description, token, private) VALUES (?, ?, ?, ?, '0')", r.ID, r.Privileges, data.Description, tokenMD5)
	if err != nil {
		md.Err(err)
		return Err500
	}

	r.Code = 200
	return r
}

// TokenSelfDeleteGET deletes the token the user is connecting with.
func TokenSelfDeleteGET(md common.MethodData) common.CodeMessager {
	if md.ID() == 0 {
		return common.SimpleResponse(400, "How should we delete your token if you haven't even given us one?!")
	}
	_, err := md.DB.Exec("DELETE FROM tokens WHERE token = ? LIMIT 1",
		fmt.Sprintf("%x", md5.Sum([]byte(md.User.Value))))
	if err != nil {
		md.Err(err)
		return Err500
	}
	return common.SimpleResponse(200, "Bye!")
}

type token struct {
	ID          int    `json:"id"`
	Privileges  uint64 `json:"privileges"`
	Description string `json:"description"`
}
type tokenResponse struct {
	common.ResponseBase
	Tokens []token `json:"token"`
}

// TokenGET retrieves a list listing all the user's public tokens.
func TokenGET(md common.MethodData) common.CodeMessager {
	rows, err := md.DB.Query("SELECT id, privileges, description FROM tokens WHERE user = ? AND private = '0'", md.ID())
	if err != nil {
		return Err500
	}
	var r tokenResponse
	for rows.Next() {
		var t token
		err = rows.Scan(&t.ID, &t.Privileges, &t.Description)
		if err != nil {
			md.Err(err)
			continue
		}
		r.Tokens = append(r.Tokens, t)
	}
	r.Code = 200
	return r
}

type tokenSingleResponse struct {
	common.ResponseBase
	token
}

// TokenSelfGET retrieves information about the token the user is connecting with.
func TokenSelfGET(md common.MethodData) common.CodeMessager {
	var r tokenSingleResponse
	// md.User.ID = token id, userid would have been md.User.UserID. what a clusterfuck
	err := md.DB.QueryRow("SELECT id, privileges, description FROM tokens WHERE id = ?", md.User.ID).Scan(
		&r.ID, &r.Privileges, &r.Description,
	)
	if err != nil {
		md.Err(err)
		return Err500
	}
	r.Code = 200
	return r
}

// TokenFixPrivilegesGET fixes the privileges on the token of the given user,
// or of all the users if no user is given.
func TokenFixPrivilegesGET(md common.MethodData) common.CodeMessager {
	id := common.Int(md.C.Query("id"))
	if md.C.Query("id") == "self" {
		id = md.ID()
	}
	go fixPrivileges(id, md.DB)
	return common.SimpleResponse(200, "Privilege fixing started!")
}

func fixPrivileges(user int, db *sql.DB) {
	var wc string
	var params = make([]interface{}, 0, 1)
	if user != 0 {
		// dirty, but who gives a shit
		wc = "WHERE user = ?"
		params = append(params, user)
	}
	rows, err := db.Query(`
SELECT
	tokens.id, tokens.privileges, users.rank
FROM tokens
LEFT JOIN users ON users.id = tokens.user
`+wc, params...)
	if err != nil {
		fmt.Println(err)
		schiavo.Bunker.Send(err.Error())
		return
	}
	for rows.Next() {
		var (
			id       int
			privsRaw uint64
			privs    common.Privileges
			newPrivs common.Privileges
			rank     int
		)
		rows.Scan(&id, &privsRaw, &rank)
		privs = common.Privileges(privsRaw)
		newPrivs = privs.CanOnly(rank)
		if newPrivs != privs {
			_, err := db.Exec("UPDATE tokens SET privileges = ? WHERE id = ? LIMIT 1", uint64(newPrivs), id)
			if err != nil {
				fmt.Println(err)
				schiavo.Bunker.Send(err.Error())
				continue
			}
		}
	}
}
