// SPDX-License-Identifier: GPL-2.0-or-later
/*
 * Copyright (C) 2018-2025 SCANOSS.COM
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

package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	purlHelper "github.com/scanoss/go-purl-helper/pkg"
	"go.uber.org/zap"
	"scanoss.com/semgrep/pkg/dtos"
	"scanoss.com/semgrep/pkg/models"
	"scanoss.com/semgrep/pkg/utils"
)

type SemgrepUseCase struct {
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
	SelectedURLS    []models.AllURL
}

func NewSemgrep(db *sqlx.DB) *SemgrepUseCase {
	return &SemgrepUseCase{
		allUrls: models.NewAllURLModel(db, models.NewProjectModel(db)),
	}
}

// GetIssues takes the Semgrep Input request, searches for Semgrep usages and returns a SemgrepOutput struct.
func (d SemgrepUseCase) GetIssues(ctx context.Context, s *zap.SugaredLogger, components []dtos.ComponentDTO) (dtos.SemgrepOutput, error) {
	query := []InternalQuery{}
	purlsToQuery := []utils.PurlReq{}
	// Prepare purls to query
	for _, c := range components {
		purlReq := strings.Split(c.Purl, "@") // Remove any version specific info from the PURL
		if purlReq[0] == "" {
			continue
		}
		if len(purlReq) > 1 {
			c.Requirement = purlReq[1]
		}

		purlName, err := purlHelper.PurlNameFromString(c.Purl) // Make sure we just have the bare minimum for a Purl Name
		if err == nil {
			purlsToQuery = append(purlsToQuery, utils.PurlReq{Purl: purlName, Version: c.Requirement})
		}
		query = append(query, InternalQuery{CompletePurl: c.Purl, Requirement: c.Requirement, PurlName: purlName})
	}

	url, err := d.allUrls.GetUrlsByPurlList(ctx, s, purlsToQuery)
	_ = err

	purlMap := make(map[string][]models.AllURL)

	///Order Urls in a map for fast access by purlname
	for r := range url {
		purlMap[url[r].PurlName] = append(purlMap[url[r].PurlName], url[r])
	}
	urlHashes := []string{}
	// For all the requested purls, choose the closest urls that match
	for r := range query {
		query[r].SelectedURLS, _ = models.PickClosestUrls(purlMap[query[r].PurlName], query[r].PurlName, "", query[r].Requirement)
		if len(query[r].SelectedURLS) > 0 {
			query[r].SelectedVersion = query[r].SelectedURLS[0].Version
			for h := range query[r].SelectedURLS {
				urlHashes = append(urlHashes, query[r].SelectedURLS[h].URLHash)
			}
		}
	}
	// Create a map containing the files for each url
	files := models.QueryBulkPivotLDB(urlHashes)

	filesURL := []string{}

	// Create a map containing the Semgrep issue for each file
	semgrep := models.QueryBulkSemgrepLDB(files)

	retV := dtos.SemgrepOutput{}

	// Create the response
	for r := range query {
		var semgrepOutItem dtos.SemgrepOutputItem

		// issues := []dtos.IssueItem{}
		relatedURLs := query[r].SelectedURLS
		semgrepOutItem.Version = query[r].SelectedVersion
		semgrepOutItem.Purl = query[r].CompletePurl
		for u := range relatedURLs {
			hash := relatedURLs[u].URLHash
			filesInURL := files[hash]
			for f := range filesInURL {
				if len(semgrep[filesInURL[f]]) > 0 {
					fileIssues := dtos.SemgrepFileIssues{File: filesInURL[f]}
					filesURL = append(filesURL, fmt.Sprintf("%s-%s", filesInURL[f], hash))
					for i := range semgrep[filesInURL[f]] {
						issue := semgrep[filesInURL[f]][i]
						fileIssues.Issues = append(fileIssues.Issues, dtos.IssueItem{RuleID: issue.RuleID, From: issue.From, To: issue.To, Severity: issue.Severity})
					}
					semgrepOutItem.Files = append(semgrepOutItem.Files, fileIssues)
				}
			}
		}
		paths := models.QueryBulkFileLDB(filesURL)
		for f := range semgrepOutItem.Files {
			key := semgrepOutItem.Files[f].File
			semgrepOutItem.Files[f].Path = paths[key]
			fmt.Println(semgrepOutItem.Files[f].Path)
		}
		retV.Purls = append(retV.Purls, semgrepOutItem)
	}

	return retV, nil
}
