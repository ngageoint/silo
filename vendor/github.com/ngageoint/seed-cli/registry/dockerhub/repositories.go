package dockerhub

import (
	"strings"

	"github.com/ngageoint/seed-cli/util"
	"github.com/ngageoint/seed-cli/objects"
)

type repositoriesResponse struct {
	Count    int
	Next     string
	Previous string
	Results  []Result
}

//Result struct representing JSON result
type Result struct {
	Name string
}

//Repositories Returns seed repositories for the given user/organization
func (registry *DockerHubRegistry) Repositories(user string) ([]string, error) {
	url := registry.url("/v2/repositories/%s/", user)
	repos := make([]string, 0, 10)
	var err error //We create this here, otherwise url will be rescoped with :=
	var response repositoriesResponse
	for err == nil {
		response.Next = ""
		url, err = registry.getDockerHubPaginatedJson(url, &response)
		for _, r := range response.Results {
			if !strings.HasSuffix(r.Name, "-seed") {
				continue
			}
			repos = append(repos, r.Name)
		}
	}
	if err != ErrNoMorePages {
		return nil, err
	}
	return repos, nil
}

//Tags Returns tags for a given user/organization and repository
func (registry *DockerHubRegistry) Tags(user, repository string) ([]string, error) {
	url := registry.url("/v2/repositories/%s/%s/tags", user, repository)
	tags := make([]string, 0, 10)
	var err error //We create this here, otherwise url will be rescoped with :=
	var response repositoriesResponse
	for err == nil {
		response.Next = ""
		url, err = registry.getDockerHubPaginatedJson(url, &response)
		for _, r := range response.Results {
			tags = append(tags, r.Name)
		}
	}
	if err != ErrNoMorePages {
		return nil, err
	}
	return tags, nil
}

//Images returns seed images for a given user/repository.  It will grab all of the seed repositories and combine them
//with any tags it can find to build a list of images.
func (registry *DockerHubRegistry) Images(user string) ([]string, error) {
	url := registry.url("/v2/repositories/%s/", user)
	registry.Print( "Searching %s for Seed images...\n", url)
	repos := make([]string, 0, 10)
	var err error //We create this here, otherwise url will be rescoped with :=
	var response repositoriesResponse
	for err == nil {
		response.Next = ""
		url, err = registry.getDockerHubPaginatedJson(url, &response)
		for _, r := range response.Results {
			if !strings.HasSuffix(r.Name, "-seed") {
				continue
			}
			// Add all tags if found
			if rs, _ := registry.Tags(user, r.Name); len(rs) > 0 {
				for _, tag := range rs {
					img := r.Name+":"+tag
					repos = append(repos, img)
				}
				// No tags found - so just add the repo name
			} else {
				repos = append(repos, r.Name)
			}
		}
	}
	if err != ErrNoMorePages {
		return nil, err
	}
	return repos, nil
}

func (registry *DockerHubRegistry) ImagesWithManifests(org string) ([]objects.Image, error) {
	imageNames, err := registry.Images(org)

	if err != nil {
		return nil, err
	}

	images := []objects.Image{}

	url := "docker.io"
	username := ""
	password := ""

	for _, imgstr := range imageNames {
		manifest := ""
		//TODO: find better, lightweight way to get manifest on low side
		imageName, err := util.DockerPull(imgstr, url, org, username, password)
		if err == nil {
			manifest, err = util.GetSeedManifestFromImage(imageName)
		}
		if err != nil {
			registry.Print("ERROR: Could not get manifest: %s\n", err.Error())
		}

		imageStruct := objects.Image{Name: imgstr, Registry: url, Org: org, Manifest: manifest}
		images = append(images, imageStruct)
	}

	return images, err
}
