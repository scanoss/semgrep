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

package dtos

import (
	"encoding/json"
	"errors"
	"fmt"

	zlog "github.com/scanoss/zap-logging-helper/pkg/logger"
)

type SemgrepOutput struct {
	Purls []SemgrepOutputItem `json:"purls"`
}

type SemgrepOutputItem struct {
	Purl    string              `json:"purl"`
	Version string              `json:"version"`
	Files   []SemgrepFileIssues `json:"files"`
}

type SemgrepFileIssues struct {
	File   string      `json:"fileMD5"`
	Path   string      `json:"path"`
	Issues []IssueItem `json:"issues"`
}

type IssueItem struct {
	RuleID   string `json:"ruleID"`
	From     string `json:"from"`
	To       string `json:"to"`
	Severity string `json:"severity"`
}

// ExportSemgrepOutput converts the SemgrepOutput structure to a byte array.
func ExportSemgrepOutput(output SemgrepOutput) ([]byte, error) {
	data, err := json.Marshal(output)
	if err != nil {
		zlog.S.Errorf("Parse failure: %v", err)
		return nil, errors.New("failed to produce JSON from semgrep output data")
	}
	return data, nil
}

// ParseSengreoOutput converts the input byte array to a SemgrepOutput structure.
func ParseSemgrepOutput(input []byte) (SemgrepOutput, error) {
	if len(input) == 0 {
		return SemgrepOutput{}, fmt.Errorf("no output Semgrep data supplied to parse")
	}
	var data SemgrepOutput
	err := json.Unmarshal(input, &data)
	if err != nil {
		zlog.S.Errorf("Parse failure: %v", err)
		return SemgrepOutput{}, fmt.Errorf("failed to parse Semgrep output data: %v", err)
	}
	zlog.S.Debugf("Parsed data2: %v", data)
	return data, nil
}
