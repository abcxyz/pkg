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

package databasetest

// This file implements docker integration.
//
// This file is only intended to be used outside of Google. Inside of Google, this file should be
// replaced with the Google-internal version.

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql" // Force mysql driver to be included.
	dockertest "github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

var nopCloser = io.NopCloser(nil)

// start starts a docker container running a DB server. A struct is returned describing how to
// connect to container, along with a cleanup function that should be called once all tests have
// finished.
//
// The returned Closer should be called in every case, even if this function returns an error. This
// ensures that the Docker container will be cleaned up if the error occurred after the container
// was created. The Closer will never be nil.
//
// Since the startup time for database containers can be as long as 20 seconds, we share the container among
// every test. Each test should use a randomly-created database/schema name to avoid collisions
// between tests.
//
// This function installs a signal handler for SIGTERM, SIGKILL, and SIGINT that attempts to clean
// up the Docker container, then runs os.Exit(1). Since the signal handler kills the process, any
// other custom signal handlers that are installed may not get a chance to run.
//
// Docker must be installed on localhost for this to work. No environment vars are needed.
func start(conf *config) (ConnInfo, io.Closer, error) {
	portMapper, closer, err := startContainer(conf)
	if err != nil {
		return ConnInfo{}, nopCloser, err
	}

	return ConnInfo{
		Hostname:   "localhost",
		PortMapper: portMapper,
	}, closer, nil
}

// Runs a docker container and returns a function to find where ports are mapped,
// along with a cleanup function that stops the container.
func startContainer(conf *config) (func(string) string, io.Closer, error) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		return nil, nopCloser, fmt.Errorf("dockertest.NewPool(): %w", err)
	}
	container, err := runContainer(conf, pool)
	if err != nil {
		return nil, nopCloser, err
	}
	closer := newContainerCloser(pool, container)
	if err := container.Expire(uint(conf.killAfterSec)); err != nil {
		return nil, closer, fmt.Errorf("resource.Expire(): %w", err)
	}
	portMapper, err := waitUntilUp(conf, mysqlTester, pool, container)
	if err != nil {
		return nil, closer, err
	}
	return portMapper, closer, nil
}

// Starts the container and returns a Resource that points to it.
func runContainer(conf *config, pool *dockertest.Pool) (*dockertest.Resource, error) {
	// pulls an image, creates a container based on it and runs it
	container, err := pool.RunWithOptions(&conf.runOptions, func(config *docker.HostConfig) {
		config.AutoRemove = true // remove storage after container exits
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	if err != nil {
		var extraMsg string
		switch {
		case strings.Contains(err.Error(), "no such file"):
			extraMsg = `. Please install docker:
		Instructions for Debian: https://docs.docker.com/engine/install/debian/
		Instructions for Mac: https://docs.docker.com/desktop/mac/install/`
		case strings.Contains(err.Error(), "permission denied"):
			extraMsg = `. To fix this, enable sudo-less docker container creation:
					1. Run "sudo adduser $USER docker" to add your user to the docker group
					2. Reboot the machine to make the group membership effective`
		case strings.Contains(err.Error(), "404"):
			extraMsg = fmt.Sprintf(". Probably the requested tag %q does not exist as a Docker image", conf.runOptions.Tag)
		}
		return nil, fmt.Errorf("pool.Run() failed starting container: %w%s", err, extraMsg)
	}
	return container, nil
}

// waitUntilUp waits for service to be reachable.
func waitUntilUp(conf *config, tester func(*config, func(string) string) error, pool *dockertest.Pool, container *dockertest.Resource) (func(string) string, error) {
	// To get the exported TCP port number for the server, we have to wait for the docker container to
	// actually start, then get the mapped port number.
	pool.MaxWait = time.Minute
	if err := pool.Retry(func() error {
		if conf.waitForPort != "" {
			port := container.GetPort(conf.waitForPort)
			if port == "" {
				return fmt.Errorf("resource.GetPort() returned empty string, container isn't ready yet")
			}
		}

		if err := tester(conf, container.GetPort); err != nil {
			conf.progressLogger.Printf("Database isn't ready yet: %v", err)
			return fmt.Errorf("Database isn't ready yet: %w", err)
		}

		conf.progressLogger.Printf("The database container is fully up and healthy")
		return nil
	}); err != nil {
		return nil, fmt.Errorf("failed to connect to database within timeout. The final attempt returned: %w", err)
	}
	return container.GetPort, nil
}

type containerCloser struct {
	once      sync.Once
	pool      *dockertest.Pool
	container *dockertest.Resource
}

func newContainerCloser(pool *dockertest.Pool, container *dockertest.Resource) *containerCloser {
	return &containerCloser{
		pool:      pool,
		container: container,
	}
}

func (p *containerCloser) Close() error {
	var err error
	p.once.Do(func() {
		err = p.pool.Purge(p.container)
	})
	if err != nil {
		return fmt.Errorf("failed stopping dabase docker container: %w", err)
	}
	return nil
}
