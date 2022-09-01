// Copyright 2022 The Authors (see AUTHORS file)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package mysqltest

// This file implements docker integration. This file should only be used outside of Google's build
// system.

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"

	_ "github.com/go-sql-driver/mysql" // Force mysql driver to be included.
	dockertest "github.com/ory/dockertest/v3"
)

var (
	once    sync.Once
	port    int
	stopper Stopper
	err     error
)

const (
	// It's OK to hardcode the root password because only boilerplate test data is stored. Also,
	// having a well-known password can help with human inspection for debugging. The value chosen for
	// the password is arbitrary. It can be changed without breaking anything; it's not hardcoded into
	// the docker image or anything like that.
	password = "8mo5lfYKjy6ebTK"

	// Containers might not be shutdown if the test terminates abnormally, such as when ctrl-C is
	// pressed during a test. Therefore we instruct the container to kill itself after a while. The
	// duration must be longer than longest test that uses MySQL (currently about 30 seconds), and
	// should also be longer than the bazel test timeout (currently about 1 minute). There's no harm
	// in leaving lots of extra time.
	mySQLContainerDeadlineSec = 10 * 60
)

// start starts a docker container running a MySQL server. A struct is returned describing how to
// connect to MySQL, along with a cleanup function that should be called once all tests have
// finished.
//
// The returned Stopper should be called in every case, even if this function returns an error. This
// ensures that the Docker container will be cleaned up if the error occurred after the container
// was created. The Stopper will never be nil.
//
// Since the startup time for this MySQL container is about 20 seconds, we share the container among
// every test. Each test should use a randomly-created database/schema name to avoid collisions
// between tests.
//
// This function installs a signal handler for SIGTERM, SIGKILL, and SIGINT that attempts to clean
// up the Docker container, then runs os.Exit(1). Since the signal handler kills the process, any
// other custom signal handlers that are installed may not get a chance to run.
//
// Docker must be installed on localhost for this to work. No environment vars are needed.
func start(conf *config) (ConnInfo, Stopper, error) {
	port, stopper, err := startContainer(conf)
	if err != nil {
		return ConnInfo{}, noOpStopper, err
	}

	return ConnInfo{
		Username: "root",
		Password: password,
		Hostname: "localhost",
		Port:     port,
	}, stopper, nil
}

// Runs a MySQL docker container and returns the TCP port number for its MySQL port, along with a
// cleanup function that stops the container.
func startContainer(conf *config) (int, Stopper, error) {
	// The following is based on the example from https://github.com/ory/dockertest#using-dockertest.
	pool, err := dockertest.NewPool("")
	if err != nil {
		return 0, noOpStopper, fmt.Errorf("dockertest.NewPool(): %w", err)
	}

	// pulls an image, creates a container based on it and runs it
	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "mysql",
		Tag:        conf.mySQLVersion,
		Env:        []string{"MYSQL_ROOT_PASSWORD=" + password},
		Entrypoint: []string{"/usr/bin/timeout", strconv.Itoa(conf.killAfterSec), "docker-entrypoint.sh"},
		Cmd:        []string{"mysqld"},
	})
	if err != nil {
		var extraMsg string
		switch {
		case strings.Contains(err.Error(), "no such file"):
			extraMsg = `. Please install docker: 
		Instructions for gLinux: https://g3doc.corp.google.com/cloud/containers/g3doc/glinux-docker/install.md
		Instructions for Debian: https://docs.docker.com/engine/install/debian/
		Instructions for Mac: https://docs.docker.com/desktop/mac/install/`
		case strings.Contains(err.Error(), "permission denied"):
			extraMsg = `. To fix this, enable sudo-less docker container creation:
					1. Run "sudo adduser $USER docker" to add your user to the docker group
					2. Reboot the machine to make the group membership effective`
		}
		return 0, noOpStopper, fmt.Errorf("pool.Run() failed starting mysql container: %w%s", err, extraMsg)
	}

	var stopOnce sync.Once // only the first call to the stopper will have an effect.
	stop := func() error {
		var err error
		stopOnce.Do(func() {
			err = pool.Purge(resource)
		})
		if err != nil {
			return fmt.Errorf("failed stopping MySQL docker container: %w", err)
		}
		return nil
	}

	// To get the TCP port number for the mysql server, we have to wait for the docker container to
	// actually start, then get the mapped port number.
	var outPort int

	// Repeatedly try to connect to the container until it's up.
	if err := pool.Retry(func() error {
		p, err := fetchPort(resource)
		if err != nil {
			return err
		}

		if err := tryLogin(p); err != nil {
			msg := fmt.Sprintf("MySQL isn't ready yet: %v", err)
			log.Print(msg)
			return errors.New(msg)
		}

		outPort = p
		return nil
	}); err != nil {
		return 0, stop, fmt.Errorf("Could not connect to database within timeout. The final attempt returned: %w", err)
	}

	return outPort, stop, nil
}

func fetchPort(r *dockertest.Resource) (int, error) {
	portStr := r.GetPort("3306/tcp")
	if portStr == "" {
		return 0, fmt.Errorf("resource.GetPort() returned empty string, container isn't ready yet")
	}
	port, err = strconv.Atoi(portStr)
	if err != nil || port <= 0 {
		return 0, fmt.Errorf("Internal error: malformed response from GetPort(): %q. "+
			"Wanted a string containing an integer.", portStr)
	}

	return port, nil
}

func tryLogin(port int) error {
	// Disabling TLS is OK because no sensitive data is exchanged, only test data.
	addr := fmt.Sprintf("root:%s@tcp(localhost:%d)/mysql?tls=false", password, port)

	log.Printf(`Checking if MySQL is up yet on localhost at %d. It's normal to see "unexpected EOF" output while it's starting.`, port)
	db, err := sql.Open("mysql", addr)
	if err != nil {
		return fmt.Errorf("sql.Open(): %w", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		return fmt.Errorf("db.Ping(): %w", err)
	}
	log.Printf("The MySQL container is fully up and healthy")

	return nil
}

func noOpStopper() error {
	return nil
}
