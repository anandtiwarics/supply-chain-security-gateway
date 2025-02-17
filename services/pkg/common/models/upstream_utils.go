package models

import (
	"errors"
	"fmt"
	"strings"

	"github.com/abhisek/supply-chain-gateway/services/pkg/common/utils"
)

var (
	errIncorrectPrefix           = errors.New("incorrect path prefix")
	errIncorrectMaven2Path       = errors.New("incorrect maven2 path")
	errIncorrectPypiPath         = errors.New("incorrect pypi path")
	errUnimplementedUpstreamType = errors.New("path resolver for upstream type is not implemented")
)

func GetUpstreamByHostAndPath(upstreams []ArtefactUpStream, host, path string) (ArtefactUpStream, error) {
	for _, upstream := range upstreams {
		if upstream.MatchHost(host) && upstream.MatchPath(path) {
			return upstream, nil
		}
	}

	return ArtefactUpStream{}, fmt.Errorf("no upstream resolved using %s/%s", host, path)
}

func GetArtefactByHostAndPath(upstreams []ArtefactUpStream, host, path string) (Artefact, error) {
	for _, upstream := range upstreams {
		if upstream.MatchHost(host) && upstream.MatchPath(path) {
			return upstream.Path2Artefact(path)
		}
	}

	return Artefact{}, fmt.Errorf("no artefact resolved using %s/%s", host, path)
}

func (s ArtefactUpStream) NeedAuthentication() bool {
	return s.Authentication.Type != ArtefactUpstreamAuthTypeNoAuth
}

func (s ArtefactUpStream) NeedUpstreamAuthentication() bool {
	return s.Repository.Authentication.Type != ArtefactUpstreamAuthTypeNoAuth
}

func (s ArtefactUpStream) MatchHost(host string) bool {
	return (s.RoutingRule.Host == host)
}

func (s ArtefactUpStream) MatchPath(path string) bool {
	path = utils.CleanPath(path)
	return strings.HasPrefix(path, s.RoutingRule.Prefix)
}

// Resolve an HTTP request path for this artefact into an Artefact model
func (s ArtefactUpStream) Path2Artefact(path string) (Artefact, error) {
	path = utils.CleanPath(path)
	if !strings.HasPrefix(path, s.RoutingRule.Prefix) {
		return Artefact{}, errIncorrectPrefix
	}

	path = strings.TrimPrefix(path, s.RoutingRule.Prefix)
	if path != "" && path[0] == '/' {
		path = path[1:]
	}

	parts := strings.Split(path, "/")
	switch s.Type {
	case ArtefactSourceTypeMaven2:
		return artefactForMaven2(parts)
	case ArtefactSourceTypePypi:
		return artefactForPypi(parts)
	default:
		return Artefact{}, errUnimplementedUpstreamType
	}
}

func artefactForPypi(parts []string) (Artefact, error) {
	if len(parts) == 0 {
		return Artefact{}, errIncorrectPypiPath
	}

	if (parts[0] == "simple") && (len(parts) >= 2) {
		parts = parts[1:]
	}

	name := parts[0]
	version := ""

	if len(parts) > 1 {
		version = parts[1]
	}

	return NewArtefact(ArtefactSource{Type: ArtefactSourceTypePypi},
		name, "", version), nil
}

func artefactForMaven2(parts []string) (Artefact, error) {
	if len(parts) < 4 {
		return Artefact{}, errIncorrectMaven2Path
	}

	// Ignore the filename
	_ = parts[:len(parts)-1]

	version := parts[len(parts)-2]
	name := parts[len(parts)-3]

	parts = parts[:len(parts)-3]
	group := strings.Join(parts, ".")

	return NewArtefact(ArtefactSource{Type: ArtefactSourceTypeMaven2},
		name, group, version), nil
}
