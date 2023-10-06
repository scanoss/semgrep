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
/*
known purls:
 {
    "purls": [
        {
            "purl": "pkg:github/movingblocks/gestalt",
            "requirement": "7.1.0"
        },
        {
            "purl": "pkg:github/apache/axis2-java",
            "requirement": "v2-m2"
        },
        {
            "purl": "pkg:gitee/echomyecho/druid-release"
        },
        {
            "purl": "pkg:github/eclipse/eclipse.platform.debug",
            "requirement": "v200812050-1330"
        }
    ]
}*/
package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	"scanoss.com/semgrep/pkg/dtos"
	zlog "scanoss.com/semgrep/pkg/logger"
	"scanoss.com/semgrep/pkg/models"
	"scanoss.com/semgrep/pkg/utils"
)

type SemgrepUseCase struct {
	ctx     context.Context
	conn    *sqlx.Conn
	allUrls *models.AllUrlsModel
}
type SemgrepWorkerStruct struct {
	URLMd5  string
	Purl    string
	Version string
}
type InternalQuery struct {
	CompletePurl    string
	PurlName        string
	Requirement     string
	SelectedVersion string
	SelectedURLS    []models.AllUrl
}

func NewSemgrep(ctx context.Context, conn *sqlx.Conn) *SemgrepUseCase {
	return &SemgrepUseCase{ctx: ctx, conn: conn,
		allUrls: models.NewAllUrlModel(ctx, conn, models.NewProjectModel(ctx, conn)),
	}
}

// GetIssues takes the Semgrep Input request, searches for Semgrep usages and returns a SemgrepOutput struct
func (d SemgrepUseCase) GetIssues(request dtos.SemgrepInput) (dtos.SemgrepOutput, error) {

	if len(request.Purls) == 0 {
		zlog.S.Info("Empty List of Purls supplied")
	}
	if len(request.Purls) == 0 {
		zlog.S.Info("Empty List of Purls supplied")
	}
	query := []InternalQuery{}
	purlsToQuery := []utils.PurlReq{}

	//Prepare purls to query
	for _, purl := range request.Purls {

		purlReq := strings.Split(purl.Purl, "@") // Remove any version specific info from the PURL
		if purlReq[0] == "" {
			continue
		}
		if len(purlReq) > 1 {
			purl.Requirement = purlReq[1]
		}

		purlName, err := utils.PurlNameFromString(purl.Purl) // Make sure we just have the bare minimum for a Purl Name
		if err == nil {
			purlsToQuery = append(purlsToQuery, utils.PurlReq{Purl: purlName, Version: purl.Requirement})
		}
		query = append(query, InternalQuery{CompletePurl: purl.Purl, Requirement: purl.Requirement, PurlName: purlName})
	}

	url, err := d.allUrls.GetUrlsByPurlList(purlsToQuery)
	_ = err

	purlMap := make(map[string][]models.AllUrl)

	///Order Urls in a map for fast access by purlname
	for r := range url {
		purlMap[url[r].PurlName] = append(purlMap[url[r].PurlName], url[r])
	}
	urlHashes := []string{}
	// For all the requested purls, choose the closest urls that match
	for r := range query {
		query[r].SelectedURLS, err = models.PickClosestUrls(purlMap[query[r].PurlName], query[r].PurlName, "", query[r].Requirement)
		if len(query[r].SelectedURLS) > 0 {
			query[r].SelectedVersion = query[r].SelectedURLS[0].Version
			for h := range query[r].SelectedURLS {
				urlHashes = append(urlHashes, query[r].SelectedURLS[h].UrlHash)
			}
		}
	}
	//Create a map containing the files for each url
	files := models.QueryBulkPivotLDB(urlHashes)

	filesUrl := []string{}

	//Create a map containing the Semgrep issue for each file
	semgrep := models.QueryBulkSemgrepLDB(files)

	/*mapSemgrep := make(map[string][]models.SemgrepItem)

	//Remove duplicate algorithms for the same file
	for k, v := range files {
		for f := range v {
			mapSemgrep[k] = append(mapSemgrep[k], semgrep[v[f]]...)
		}
	}*/
	retV := dtos.SemgrepOutput{}

	//Create the response
	for r := range query {
		var semgrepOutItem dtos.SemgrepOutputItem

		//issues := []dtos.IssueItem{}
		relatedURLs := query[r].SelectedURLS
		semgrepOutItem.Version = query[r].SelectedVersion
		semgrepOutItem.Purl = query[r].CompletePurl
		for u := range relatedURLs {
			hash := relatedURLs[u].UrlHash
			filesInUrl := files[hash]
			for f := range filesInUrl {
				if len(semgrep[filesInUrl[f]]) > 0 {
					fileIssues := dtos.SemgrepFileIssues{File: filesInUrl[f]}
					filesUrl = append(filesUrl, fmt.Sprintf("%s-%s", filesInUrl[f], hash))
					for i := range semgrep[filesInUrl[f]] {
						issue := semgrep[filesInUrl[f]][i]
						fileIssues.Issues = append(fileIssues.Issues, dtos.IssueItem{RuleID: issue.RuleID, From: issue.From, To: issue.To, Severity: issue.Severity})
					}
					semgrepOutItem.Files = append(semgrepOutItem.Files, fileIssues)
				}
			}
		}
		paths := models.QueryBulkFileLDB(filesUrl)
		for f := range semgrepOutItem.Files {
			key := semgrepOutItem.Files[f].File
			semgrepOutItem.Files[f].Path = paths[key]
			fmt.Println(semgrepOutItem.Files[f].Path)
		}
		retV.Purls = append(retV.Purls, semgrepOutItem)

	}

	return retV, nil
}
