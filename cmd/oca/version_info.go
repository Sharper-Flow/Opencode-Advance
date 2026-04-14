package main

var version = ""

const defaultVersion = "0.0.0-dev"

type VersionInfo struct {
	Version string
}

func CurrentVersionInfo() VersionInfo {
	return VersionInfo{Version: version}
}

func (v VersionInfo) Display() string {
	if v.Version == "" {
		return defaultVersion
	}
	return v.Version
}
