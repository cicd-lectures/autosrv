package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	docker "github.com/docker/docker/client"
)

type Envelope struct {
	Events []Event `json:"events"`
}

type Event struct {
	ID      string  `json:"id"`
	Action  string  `json:"action"`
	Target  Target  `json:"target"`
	Request Request `json:"Request"`
}

type Target struct {
	Repository string `json:"repository"`
	Tag        string `json:"Tag"`
	Digest     string `json:"digest"`
}

type Request struct {
	Host string `json:"host"`
}

type Deployer interface {
	Deploy(context.Context, Target, Request) error
}

func handleNotification(d Deployer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println("Receiving notification")
		defer r.Body.Close()

		var e Envelope
		if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
			log.Println("Failed to decode body:", err)
			http.Error(w, "unable to decode body", http.StatusInternalServerError)
			return
		}

		for _, evt := range e.Events {
			switch evt.Action {
			case "push":
				log.Printf("Received a push %q, %q, %q", evt.Target.Repository, evt.Target.Tag, evt.Target.Digest)
				// Sanitize event.
				if evt.Target.Tag == "" {
					evt.Target.Tag = "latest"
				}

				if err := d.Deploy(r.Context(), evt.Target, evt.Request); err != nil {
					log.Printf("Unable to deploy target: %v", err)
					http.Error(w, "unable to deploy target", http.StatusInternalServerError)
					return
				}

			default:
				log.Printf("Unmanaged action %q, ignoring", evt.Action)
			}
		}

		w.WriteHeader(http.StatusOK)
	}
}

type DockerDeployer struct {
	docker         Docker
	routingNetwork types.NetworkResource
}

func NewDockerDeployer(d Docker, net types.NetworkResource) *DockerDeployer {
	return &DockerDeployer{docker: d, routingNetwork: net}
}

type Docker interface {
	ContainerList(context.Context, types.ContainerListOptions) ([]types.Container, error)
	ContainerCreate(context.Context, *container.Config, *container.HostConfig, *network.NetworkingConfig, string) (container.ContainerCreateCreatedBody, error)
	ContainerStart(ctx context.Context, containerID string, opts types.ContainerStartOptions) error

	ContainerStop(ctx context.Context, containerID string, timeout *time.Duration) error
	ContainerRemove(ctx context.Context, containerID string, options types.ContainerRemoveOptions) error

	ImageInspectWithRaw(ctx context.Context, imgID string) (types.ImageInspect, []byte, error)
	ImagePull(ctx context.Context, refStr string, options types.ImagePullOptions) (io.ReadCloser, error)
}

func (d DockerDeployer) Deploy(ctx context.Context, t Target, r Request) error {
	// Evaluate container name.
	cName := strings.ReplaceAll(t.Repository, "/", "_")

	// Find any existing container running carrying the normalized name.
	olds, err := d.docker.ContainerList(ctx, types.ContainerListOptions{
		Filters: filters.NewArgs(filters.Arg("name", cName)),
	})
	if err != nil {
		return fmt.Errorf("unable to list containers: %w", err)
	}

	for _, c := range olds {
		// if exist, check the image sha, and if we find at least one container
		// running this image then we don't have anything to to do.
		img, _, err := d.docker.ImageInspectWithRaw(ctx, c.ImageID)
		if err != nil {
			return fmt.Errorf("unable to collect image info: %w", err)
		}

		for _, digest := range img.RepoDigests {
			if strings.Contains(digest, t.Digest) {
				log.Printf("Image is already running for repo %q, nothing to do...", t.Repository)
				return nil
			}
		}

		log.Printf("New image detected for repository %q, stopping container %q...", t.Repository, c.ID)

		// Otherwise we stop any old instance of containers running the old image.
		t := time.Second
		if err = d.docker.ContainerStop(ctx, c.ID, &t); err != nil {
			return fmt.Errorf("unable to stop previous container: %w", err)
		}

		if err = d.docker.ContainerRemove(ctx, c.ID, types.ContainerRemoveOptions{}); err != nil {
			return fmt.Errorf("unable to remove previous container: %w", err)
		}
	}

	// Pull the image from the registry to the new host.
	imgRef := r.Host + "/" + t.Repository + ":" + t.Tag
	log.Println("Pulling image", imgRef)

	st, err := d.docker.ImagePull(ctx, imgRef, types.ImagePullOptions{})
	if err != nil {
		return fmt.Errorf("unable to pull image: %w", err)
	}
	defer st.Close()

	if _, err = io.Copy(ioutil.Discard, st); err != nil {
		return fmt.Errorf("unable to pull image: %w", err)
	}

	// Create and run the container.
	log.Println("Creating a new container", cName)
	c, err := d.docker.ContainerCreate(
		ctx,
		&container.Config{
			Image: imgRef,
			Labels: map[string]string{
				traefikRouterName(t.Repository) + ".rule":        traefikRouterRule(t.Repository),
				traefikRouterName(t.Repository) + ".tls":         "true",
				traefikRouterName(t.Repository) + ".entrypoints": "websecure",
				"traefik.docker.network":                         d.routingNetwork.Name,
				"traefik.enable":                                 "true",
			},
		},
		&container.HostConfig{},
		&network.NetworkingConfig{
			EndpointsConfig: map[string]*network.EndpointSettings{
				d.routingNetwork.Name: {
					NetworkID: d.routingNetwork.ID,
				},
			},
		},
		cName,
	)
	if err != nil {
		return fmt.Errorf("unable to create container: %w", err)
	}

	if err = d.docker.ContainerStart(ctx, c.ID, types.ContainerStartOptions{}); err != nil {
		return fmt.Errorf("unable to start container: %w", err)
	}

	log.Printf("Latest image for repository %q is running with container %q", t.Repository, c.ID)

	return nil
}

func traefikRouterName(repository string) string {
	return fmt.Sprintf("traefik.http.routers.%s", strings.ReplaceAll(repository, "/", ""))
}

func traefikRouterRule(repository string) string {
	return "Path(`/" + path.Join("apps", repository) + "`)"
}

func main() {
	if len(os.Args) != 2 {
		log.Fatalf("Invalid number of arguments")
	}

	log.Printf("Starting deployer connected to network %q", os.Args[1])

	c, err := docker.NewClientWithOpts(docker.FromEnv, docker.WithAPIVersionNegotiation())
	if err != nil {
		log.Fatalf("unable to create docker client: %v", err)
	}

	nets, err := c.NetworkList(context.Background(), types.NetworkListOptions{
		Filters: filters.NewArgs(filters.Arg("name", os.Args[1])),
	})
	if err != nil {
		log.Fatalf("unable to collect network information: %v", err)
	}
	if len(nets) != 1 {
		log.Fatalf("network %q is not unique on the host, please clean it up", os.Args[1])
	}

	http.Handle("/notification", http.HandlerFunc(handleNotification(NewDockerDeployer(c, nets[0]))))

	log.Println("Listening on port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
