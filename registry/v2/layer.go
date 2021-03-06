package v2

import (
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/docker/distribution"
	"github.com/docker/distribution/digest"
)

func (registry *V2registry) DownloadLayer(repository string, digest digest.Digest) (io.ReadCloser, error) {
	url := registry.url("/v2/%s/blobs/%s", repository, digest)
	// registry.Logf("registry.layer.download url=%s repository=%s digest=%s", url, repository, digest)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	// look for token here, set it here
	token, err := registry.GetOrCreateToken(repository, url)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token.Token))
	// req.SetBasicAuth(registry.Username, registry.Password)

	resp, err := registry.Client.Do(req)
	if err != nil || resp.StatusCode == http.StatusUnauthorized {
		return nil, err
	}

	return resp.Body, nil
}

func (registry *V2registry) UploadLayer(repository string, digest digest.Digest, content io.Reader) error {
	uploadUrl, err := registry.initiateUpload(repository)
	if err != nil {
		return err
	}
	q := uploadUrl.Query()
	q.Set("digest", digest.String())
	uploadUrl.RawQuery = q.Encode()

	// registry.Logf("registry.layer.upload url=%s repository=%s digest=%s", uploadUrl, repository, digest)

	upload, err := http.NewRequest("PUT", uploadUrl.String(), content)
	if err != nil {
		return err
	}
	upload.Header.Set("Content-Type", "application/octet-stream")

	_, err = registry.Client.Do(upload)
	return err
}

func (registry *V2registry) HasLayer(repository string, digest digest.Digest) (bool, error) {
	checkUrl := registry.url("/v2/%s/blobs/%s", repository, digest)
	// registry.Logf("registry.layer.check url=%s repository=%s digest=%s", checkUrl, repository, digest)

	resp, err := registry.Client.Head(checkUrl)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err == nil {
		return resp.StatusCode == http.StatusOK, nil
	}

	urlErr, ok := err.(*url.Error)
	if !ok {
		return false, err
	}
	httpErr, ok := urlErr.Err.(*HttpStatusError)
	if !ok {
		return false, err
	}
	if httpErr.Response.StatusCode == http.StatusNotFound {
		return false, nil
	}

	return false, err
}

func (registry *V2registry) LayerMetadata(repository string, digest digest.Digest) (distribution.Descriptor, error) {
	checkUrl := registry.url("/v2/%s/blobs/%s", repository, digest)
	// registry.Logf("registry.layer.check url=%s repository=%s digest=%s", checkUrl, repository, digest)

	resp, err := registry.Client.Head(checkUrl)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return distribution.Descriptor{}, err
	}

	return distribution.Descriptor{
		Digest: digest,
		Size:   resp.ContentLength,
	}, nil
}

func (registry *V2registry) initiateUpload(repository string) (*url.URL, error) {
	initiateUrl := registry.url("/v2/%s/blobs/uploads/", repository)
	// registry.Logf("registry.layer.initiate-upload url=%s repository=%s", initiateUrl, repository)

	resp, err := registry.Client.Post(initiateUrl, "application/octet-stream", nil)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return nil, err
	}

	location := resp.Header.Get("Location")
	locationUrl, err := url.Parse(location)
	if err != nil {
		return nil, err
	}
	return locationUrl, nil
}
