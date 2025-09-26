// SPDX-License-Identifier: GPL-2.0-or-later
/*
 * Copyright (C) 2018-2022 SCANOSS.COM
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 2 of the License, or
 * (at your option) any later version.
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package models

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	semver "github.com/Masterminds/semver/v3"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
	zlog "scanoss.com/semgrep/pkg/logger"

	"scanoss.com/semgrep/pkg/utils"
)

type AllUrlsModel struct {
	db      *sqlx.DB
	project *projectModel
}

type AllURL struct {
	URLHash   string `db:"url_hash"`
	Component string `db:"component"`
	Version   string `db:"version"`
	SemVer    string `db:"semver"`
	PurlName  string `db:"purl_name"`
	PurlType  string `db:"purl_type"`
	MineID    int32  `db:"mine_id"`
	URL       string `db:"-"`
}

// NewAllURLModel creates a new instance of the All URL Model.
func NewAllURLModel(db *sqlx.DB, project *projectModel) *AllUrlsModel {
	return &AllUrlsModel{db: db, project: project}
}

func (m *AllUrlsModel) GetUrlsByPurlList(ctx context.Context, s *zap.SugaredLogger, list []utils.PurlReq) ([]AllURL, error) {
	if len(list) == 0 {
		s.Errorf("Please specify a valid Purl list to query")
		return []AllURL{}, errors.New("please specify a valid Purl list to query")
	}
	purlNames := []string{}

	for p := range list {
		purlNames = append(purlNames, "'"+list[p].Purl+"'")
	}
	inStmt := strings.Join(purlNames, ",")
	inStmt = "(" + inStmt + ")"
	stmt := "SELECT package_hash AS url_hash, component, v.version_name AS version, v.semver AS semver, m.purl_type as purl_type, " +
		"purl_name, mine_id FROM all_urls u " +
		"LEFT JOIN mines m ON u.mine_id = m.id " +
		"LEFT JOIN versions v ON u.version_id = v.id " +
		"WHERE u.purl_name in " + inStmt +
		" and package_hash!= '' ORDER BY date DESC;"

	var allUrls []AllURL
	err := m.db.SelectContext(ctx, &allUrls, stmt)
	if err != nil {
		s.Errorf("Failed to query a list of urls:  %v", err)
		return []AllURL{}, fmt.Errorf("failed to query the all urls table: %v", err)
	}
	// zlog.S.Debugf("Found %v results for %v, %v.", len(allUrls), purlType, purlName)
	return allUrls, nil
}

// GetUrlsByPurlString searches for component details of the specified Purl string (and optional requirement).
func (m *AllUrlsModel) GetUrlsByPurlString(ctx context.Context, purlString, purlReq string) (AllURL, error) {
	if len(purlString) == 0 {
		zlog.S.Errorf("Please specify a valid Purl String to query")
		return AllURL{}, errors.New("please specify a valid Purl String to query")
	}
	purl, err := utils.PurlFromString(purlString)
	if err != nil {
		return AllURL{}, err
	}
	purlName, err := utils.PurlNameFromString(purlString) // Make sure we just have the bare minimum for a Purl Name
	if err != nil {
		return AllURL{}, err
	}
	if len(purl.Version) == 0 && len(purlReq) > 0 { // No version specified, but we might have a specific version in the Requirement
		ver := utils.GetVersionFromReq(purlReq)
		if len(ver) > 0 {
			// TODO check what to do if we get a "file" requirement
			purl.Version = ver // Switch to exact version search (faster)
			purlReq = ""
		}
	}

	if len(purl.Version) > 0 {
		return m.GetUrlsByPurlNameTypeVersion(ctx, purlName, purl.Type, purl.Version)
	}
	return m.GetUrlsByPurlNameType(ctx, purlName, purl.Type, purlReq)
}

// GetUrlsByPurlNameType searches for component details of the specified Purl Name/Type (and optional requirement).
func (m *AllUrlsModel) GetUrlsByPurlNameType(ctx context.Context, purlName, purlType, purlReq string) (AllURL, error) {
	if len(purlName) == 0 {
		zlog.S.Errorf("Please specify a valid Purl Name to query")
		return AllURL{}, errors.New("please specify a valid Purl Name to query")
	}
	if len(purlType) == 0 {
		zlog.S.Errorf("Please specify a valid Purl Type to query: %v", purlName)
		return AllURL{}, errors.New("please specify a valid Purl Type to query")
	}
	var allUrls []AllURL
	err := m.db.SelectContext(ctx, &allUrls,
		"SELECT package_hash AS url_hash, component, v.version_name AS version, v.semver AS semver, "+
			"purl_name, mine_id FROM all_urls u "+
			"LEFT JOIN mines m ON u.mine_id = m.id "+
			"LEFT JOIN versions v ON u.version_id = v.id "+
			"WHERE m.purl_type = $1 AND u.purl_name = $2 AND is_mined = true "+
			"ORDER BY date DESC;",
		purlType, purlName)
	if err != nil {
		zlog.S.Errorf("Failed to query all urls table for %v - %v: %v", purlType, purlName, err)
		return AllURL{}, fmt.Errorf("failed to query the all urls table: %v", err)
	}
	zlog.S.Debugf("Found %v results for %v, %v.", len(allUrls), purlType, purlName)
	// Pick one URL to return (checking for license details also)
	return pickOneURL(allUrls, purlName, purlType, purlReq)
}

// GetUrlsByPurlNameTypeVersion searches for component details of the specified Purl Name/Type and version.
func (m *AllUrlsModel) GetUrlsByPurlNameTypeVersion(ctx context.Context, purlName, purlType, purlVersion string) (AllURL, error) {
	if len(purlName) == 0 {
		zlog.S.Errorf("Please specify a valid Purl Name to query")
		return AllURL{}, errors.New("please specify a valid Purl Name to query")
	}
	if len(purlType) == 0 {
		zlog.S.Errorf("Please specify a valid Purl Type to query")
		return AllURL{}, errors.New("please specify a valid Purl Type to query")
	}
	if len(purlVersion) == 0 {
		zlog.S.Errorf("Please specify a valid Purl Version to query")
		return AllURL{}, errors.New("please specify a valid Purl Version to query")
	}
	var allUrls []AllURL
	err := m.db.SelectContext(ctx, &allUrls,
		"SELECT package_hash AS url_hash, component, v.version_name AS version, v.semver AS semver, "+
			"purl_name, mine_id FROM all_urls u "+
			"LEFT JOIN mines m ON u.mine_id = m.id "+
			"LEFT JOIN versions v ON u.version_id = v.id "+
			"WHERE m.purl_type = $1 AND u.purl_name = $2 AND v.version_name = $3 AND is_mined = true "+
			"ORDER BY date DESC;",
		purlType, purlName, purlVersion)
	if err != nil {
		zlog.S.Errorf("Failed to query all urls table for %v - %v: %v", purlType, purlName, err)
		return AllURL{}, fmt.Errorf("failed to query the all urls table: %v", err)
	}
	zlog.S.Debugf("Found %v results for %v, %v.", len(allUrls), purlType, purlName)
	// Pick one URL to return (checking for license details also)
	return pickOneURL(allUrls, purlName, purlType, "")
}

// pickOneURL takes the potential matching component/versions and selects the most appropriate one.
func pickOneURL(allUrls []AllURL, purlName, purlType, purlReq string) (AllURL, error) {
	if len(allUrls) == 0 {
		zlog.S.Infof("No component match (in urls) found for %v, %v", purlName, purlType)
		return AllURL{}, nil
	}
	zlog.S.Debugf("Potential Matches: %v", allUrls)
	var c *semver.Constraints
	var urlMap = make(map[*semver.Version]AllURL)

	if len(purlReq) > 0 {
		zlog.S.Debugf("Building version constraint for %v: %v", purlName, purlReq)
		var err error
		c, err = semver.NewConstraint(purlReq)
		if err != nil {
			zlog.S.Warnf("Encountered an issue parsing version constraint string '%v' (%v,%v): %v", purlReq, purlName, purlType, err)
		}
	}

	zlog.S.Debugf("Checking versions...")
	for _, url := range allUrls {
		if len(url.SemVer) > 0 || len(url.Version) > 0 {
			v, err := semver.NewVersion(url.Version)
			if err != nil && len(url.SemVer) > 0 {
				//	zlog.S.Debugf("Failed to parse SemVer: '%v'. Trying Version instead: %v (%v)", url.Version, url.SemVer, err)
				v, err = semver.NewVersion(url.SemVer) // Semver failed, try the normal version
			}
			if err != nil {
				zlog.S.Warnf("Encountered an issue parsing version string '%v' (%v) for %v: %v. Using v0.0.0", url.Version, url.SemVer, url, err)
				v, err = semver.NewVersion("v0.0.0") // Semver failed, just use a standard version zero (for now)
			}
			if err == nil {
				if c == nil || c.Check(v) {
					_, ok := urlMap[v]
					if !ok {
						urlMap[v] = url // fits inside the constraint and hasn't already been stored
					}
				}
			}
		} else {
			zlog.S.Warnf("Skipping match as it doesn't have a version: %#v", url)
		}
	}
	if len(urlMap) == 0 { // TODO should we return the latest version anyway?
		zlog.S.Warnf("No component match found for %v, %v after filter %v", purlName, purlType, purlReq)
		return AllURL{}, nil
	}
	var versions = make([]*semver.Version, len(urlMap))
	var vi = 0
	for version := range urlMap { // Save the list of versions so they can be sorted
		versions[vi] = version
		vi++
	}
	zlog.S.Debugf("Version List: %v", versions)
	sort.Sort(semver.Collection(versions))
	version := versions[len(versions)-1] // Get the latest (acceptable) URL version
	zlog.S.Debugf("Sorted versions: %v. Highest: %v", versions, version)

	url, ok := urlMap[version] // Retrieve the latest accepted URL version
	if !ok {
		zlog.S.Errorf("Problem retrieving URL data for %v (%v, %v)", version, purlName, purlType)
		return AllURL{}, fmt.Errorf("failed to retrieve specific URL version: %v", version)
	}
	url.URL, _ = utils.ProjectURL(purlName, purlType)

	zlog.S.Debugf("Selected version: %#v", url)
	return url, nil // Return the best component match
}

func PickClosestUrls(allUrls []AllURL, purlName, purlType, purlReq string) ([]AllURL, error) {
	if len(allUrls) == 0 {
		zlog.S.Infof("No component match (in urls) found for %v, %v", purlName, purlType)
		return []AllURL{}, nil
	}
	// zlog.S.Debugf("Potential Matches: %v", allUrls)
	var c *semver.Constraints
	var urlMap = make(map[*semver.Version][]AllURL)

	if len(purlReq) > 0 {
		zlog.S.Debugf("Building version constraint for %v: %v", purlName, purlReq)
		var err error
		c, err = semver.NewConstraint(purlReq)
		if err != nil {
			zlog.S.Warnf("Encountered an issue parsing version constraint string '%v' (%v,%v): %v", purlReq, purlName, purlType, err)
		}
	}

	zlog.S.Debugf("Checking versions...")
	for _, url := range allUrls {
		if len(url.SemVer) > 0 || len(url.Version) > 0 {
			v, err := semver.NewVersion(url.Version)
			if err != nil && len(url.SemVer) > 0 {
				//	zlog.S.Debugf("Failed to parse SemVer: '%v'. Trying Version instead: %v (%v)", url.Version, url.SemVer, err)
				v, err = semver.NewVersion(url.SemVer) // Semver failed, try the normal version
			}
			if err != nil {
				zlog.S.Warnf("Encountered an issue parsing version string '%v' (%v) for %v: %v. Using v0.0.0", url.Version, url.SemVer, url, err)
				v, err = semver.NewVersion("v0.0.0") // Semver failed, just use a standard version zero (for now)
			}
			if err == nil {
				if c == nil || c.Check(v) {
					_, ok := urlMap[v]
					if !ok {
						urlMap[v] = append(urlMap[v], url) // fits inside the constraint and hasn't already been stored
					}
				}
			}
		} else {
			zlog.S.Warnf("Skipping match as it doesn't have a version: %#v", url)
		}
	}
	if len(urlMap) == 0 { // TODO should we return the latest version anyway?
		zlog.S.Warnf("No component match found for %v, %v after filter %v", purlName, purlType, purlReq)
		return []AllURL{}, nil
	}
	var versions = make([]*semver.Version, len(urlMap))
	var vi = 0
	for version := range urlMap { // Save the list of versions so they can be sorted
		versions[vi] = version
		vi++
	}
	// zlog.S.Debugf("Version List: %v", versions)
	sort.Sort(semver.Collection(versions))
	version := versions[len(versions)-1] // Get the latest (acceptable) URL version
	// zlog.S.Debugf("Sorted versions: %v. Highest: %v", versions, version)

	url, ok := urlMap[version] // Retrieve the latest accepted URL version
	if !ok {
		zlog.S.Errorf("Problem retrieving URL data for %v (%v, %v)", version, purlName, purlType)
		return []AllURL{}, fmt.Errorf("failed to retrieve specific URL version: %v", version)
	}
	///url.URL, _ = utils.ProjectURL(purlName, purlType)

	zlog.S.Debugf("Selected version: %#v", url)
	return url, nil // Return the best component match
}
