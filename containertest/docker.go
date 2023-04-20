// Copyright 2023 The Authors (see AUTHORS file)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package containertest

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

	dockertest "github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

var nopCloser = io.NopCloser(nil)

// start starts a docker container running a service. A struct is returned describing how to
// connect to the container, along with a cleanup function that should be called once all tests have
// finished.
//
// The returned ConnInfo.Close() should be called in every case, even if this function returns an error. This
// ensures that the Docker container will be cleaned up if the error occurred after the container
// was created. The ConnInfo.Close() will never be nil.
//
// (For databases): Since the startup time for database containers can be as long as 20 seconds,
// we share the container among every test. Each test should use a randomly-created
// database/schema name to avoid collisions between tests.
//
// This function installs a signal handler for SIGTERM, SIGKILL, and SIGINT that attempts to clean
// up the Docker container, then runs os.Exit(1). Since the signal handler kills the process, any
// other custom signal handlers that are installed may not get a chance to run.
//
// Docker must be installed on localhost for this to work. No environment vars are needed.
func start(conf *config) (ConnInfo, error) {
	connInfo, err := startContainer(conf)

	// connInfo will have a closer in all cases.
	return connInfo, err
}

// Runs a docker container and returns a struct with information on exposed ports
// and host, and implements io.Closer. ConnInfo.Close() should be called regardless
// of value of error. A noop implementation is used if container hasn't started.
func startContainer(conf *config) (ConnInfo, error) {
	closerOnlyConnInfo := ConnInfo{closer: nopCloser}
	pool, err := dockertest.NewPool("")
	if err != nil {
		return closerOnlyConnInfo, fmt.Errorf("dockertest.NewPool(): %w", err)
	}
	container, err := runContainer(conf, pool)
	if err != nil {
		return closerOnlyConnInfo, err
	}

	closerOnlyConnInfo.closer = newContainerCloser(pool, container)
	if err := container.Expire(uint(conf.killAfterSec)); err != nil {
		return closerOnlyConnInfo, fmt.Errorf("resource.Expire(): %w", err)
	}
	fullConnInfo, err := waitUntilUp(conf, pool, container)
	if err != nil {
		return closerOnlyConnInfo, err
	}
	fullConnInfo.closer = closerOnlyConnInfo.closer
	return *fullConnInfo, nil
}

// Starts the container and returns a Resource that points to it.
func runContainer(conf *config, pool *dockertest.Pool) (*dockertest.Resource, error) {
	// pulls an image, creates a container based on it and runs it
	container, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: conf.service.ImageRepository(),
		Tag:        conf.service.ImageTag(),
		Env:        conf.service.Environment(),
	}, func(config *docker.HostConfig) {
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
			extraMsg = fmt.Sprintf(". Probably the requested tag %q does not exist as a Docker image", conf.service.ImageTag())
		}
		return nil, fmt.Errorf("pool.Run() failed starting container: %w%s", err, extraMsg)
	}
	return container, nil
}

// waitUntilUp waits for service to be reachable.
func waitUntilUp(conf *config, pool *dockertest.Pool, container *dockertest.Resource) (*ConnInfo, error) {
	var connInfo *ConnInfo
	// To get the exported TCP port number for the server, we have to wait for the docker container to
	// actually start, then get the mapped port number.
	pool.MaxWait = time.Minute
	if err := pool.Retry(func() error {
		notReadyPorts := make([]string, 0, len(conf.service.StartupPorts()))
		for _, waitPort := range conf.service.StartupPorts() {
			if waitPort != "" {
				port := container.GetPort(waitPort)
				if port == "" {
					notReadyPorts = append(notReadyPorts, port)
				}
			}
		}
		if len(notReadyPorts) > 0 {
			return fmt.Errorf("resource.GetPort() returned empty string for port(s) %+q, container isn't ready yet", notReadyPorts)
		}

		connInfo = &ConnInfo{
			Host:       "localhost",
			PortMapper: container.GetPort,
		}

		if err := conf.service.TestConn(conf.progressLogger, *connInfo); err != nil {
			conf.progressLogger.Printf("Container isn't ready yet: %v", err)
			return fmt.Errorf("container isn't ready yet: %w", err)
		}

		conf.progressLogger.Printf("The container is fully up and healthy")
		return nil
	}); err != nil {
		return nil, fmt.Errorf("failed to confirm container startup, final attempt returned: %w", err)
	}
	return connInfo, nil
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
		return fmt.Errorf("failed stopping docker container: %w", err)
	}
	return nil
}
