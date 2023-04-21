package containertest

import (
	"database/sql"
	"fmt"
	"net"
	"os"
	"testing"

	_ "github.com/go-sql-driver/mysql" // Link with the Go MySQL driver
)

var ci *ConnInfo
var mySQLService *MySQL

// TestMain runs once at startup to do test initialization common to all tests.
func TestMain(m *testing.M) {
	// Runs unit tests
	os.Exit(func() int {
		mySQLService = &MySQL{"5.7"}
		var err error                 // := assignment on next line would shadow ci global variable
		ci, err = Start(mySQLService) // Start the docker container. Can also pass options.
		if err != nil {
			panic(fmt.Errorf("could not start mysql service: %w", err))
		}
		defer ci.Close()

		return m.Run()
	}())
}

func TestFoo(t *testing.T) {
	t.Parallel()
	// One thing you might want to do is create an SQL driver:

	m := mySQLService
	// Find the port docker exposed for your container
	mySQLPort := ci.PortMapper(m.Port())
	uri := fmt.Sprintf("%s:%s@tcp(%s)/%s", m.Username(), m.Password(),
		net.JoinHostPort(ci.Host, mySQLPort), "")
	db, err := sql.Open("mysql", uri)
	if err != nil {
		t.Fatal(err)
	}

	// application logic goes here
	_ = db
}
