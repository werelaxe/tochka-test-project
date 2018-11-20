package main

import (
	"errors"
	"fmt"
	"github.com/fsouza/go-dockerclient"
	"log"
	"strconv"
	"time"
)

const TestConfigPath = "test.config"

func StartPostgres() (func(), error) {
	log.Println("starting postgres")
	config, err := ParseConfig(TestConfigPath)
	if err != nil {
		return nil, errors.New("test.config parsing error: " + err.Error())
	}

	endpoint := "unix:///var/run/docker.sock"
	client, _ := docker.NewClient(endpoint)

	port := docker.Port(fmt.Sprintf("%d/tcp", config.PostgresConfig.Port))
	exposedPort := map[docker.Port]struct{}{
		port: {},
	}

	createContConf := docker.Config{
		ExposedPorts: exposedPort,
		Image: "postgres",
	}

	bindPort := strconv.Itoa(int(config.PostgresConfig.Port))
	portBindings := map[docker.Port][]docker.PortBinding{
		"5432/tcp": []docker.PortBinding{docker.PortBinding{HostPort: bindPort}},
	}

	createContHostConfig := docker.HostConfig{
		PortBindings:    portBindings,
		PublishAllPorts: true,
		Privileged:      false,
	}

	createContOps := docker.CreateContainerOptions{
		Name: "test_postgres",
		Config: &createContConf,
		HostConfig: &createContHostConfig,
	}

	c, err := client.CreateContainer(createContOps)
	if err != nil {
		return nil, errors.New("can not create container: " + err.Error())
	}

	deferFn := func() {
		if err := client.RemoveContainer(docker.RemoveContainerOptions{
			ID:    c.ID,
			Force: true,
		}); err != nil {
			log.Printf("cannot remove container: %s\n", err.Error())
		}
	}
	client.StartContainer(c.ID, &docker.HostConfig{})
	if err != nil {
		deferFn()
		return nil, errors.New("can not start container: " + err.Error())
	}
	if err := waitStarted(client, c.ID, time.Second*10); err != nil {
		deferFn()
		return nil, errors.New("can not wait container: " + err.Error())
	}

	if err := waitPostgres(&config.PostgresConfig, time.Second*10); err != nil {
		deferFn()
		return nil, errors.New("can not wait database accessibility: " + err.Error())
	}
	return deferFn, nil
}

func waitStarted(client *docker.Client, id string, maxWait time.Duration) error {
	done := time.Now().Add(maxWait)
	for time.Now().Before(done) {
		c, err := client.InspectContainer(id)
		if err != nil {
			break
		}
		if c.State.Running {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return fmt.Errorf("cannot start container %s for %v", id, maxWait)
}

func waitPostgres(pconf *PostgresConfig, maxWait time.Duration) error {
	done := time.Now().Add(maxWait)
	for time.Now().Before(done) {
		err := CreateDbIfNotExists(pconf)
		if err == nil {
			return nil
		} else {
			log.Println("can not connect to database: " + err.Error())
		}
		time.Sleep(1000 * time.Millisecond)
	}
	return fmt.Errorf("cannot connect to database via %s:%d with config %v", pconf.Host, pconf.Port, *pconf)
}
