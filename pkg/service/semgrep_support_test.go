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

package service

import (
	"reflect"
	"testing"

	common "github.com/scanoss/papi/api/commonv2"
	"scanoss.com/semgrep/pkg/dtos"
)

func TestConvertSemgrepInput(t *testing.T) {
	tests := []struct {
		name    string
		request *common.PurlRequest
		want    []dtos.ComponentDTO
		wantErr bool
		errType string
	}{
		{
			name: "Valid request with single purl",
			request: &common.PurlRequest{
				Purls: []*common.PurlRequest_Purls{
					{
						Purl:        "pkg:npm/lodash",
						Requirement: "^4.17.0",
					},
				},
			},
			want: []dtos.ComponentDTO{
				{
					Purl:        "pkg:npm/lodash",
					Requirement: "^4.17.0",
				},
			},
			wantErr: false,
		},
		{
			name: "Valid request with multiple purls",
			request: &common.PurlRequest{
				Purls: []*common.PurlRequest_Purls{
					{
						Purl:        "pkg:npm/lodash",
						Requirement: "^4.17.0",
					},
					{
						Purl:        "pkg:npm/react",
						Requirement: "^18.0.0",
					},
				},
			},
			want: []dtos.ComponentDTO{
				{
					Purl:        "pkg:npm/lodash",
					Requirement: "^4.17.0",
				},
				{
					Purl:        "pkg:npm/react",
					Requirement: "^18.0.0",
				},
			},
			wantErr: false,
		},
		{
			name:    "Nil request",
			request: nil,
			want:    []dtos.ComponentDTO{},
			wantErr: true,
			errType: "BadRequestError",
		},
		{
			name: "Request with nil purls",
			request: &common.PurlRequest{
				Purls: nil,
			},
			want:    []dtos.ComponentDTO{},
			wantErr: true,
			errType: "BadRequestError",
		},
		{
			name: "Request with empty purls",
			request: &common.PurlRequest{
				Purls: []*common.PurlRequest_Purls{},
			},
			want:    []dtos.ComponentDTO{},
			wantErr: true,
			errType: "BadRequestError",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := convertSemgrepInput(tt.request)

			if (err != nil) != tt.wantErr {
				t.Errorf("convertSemgrepInput() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("convertSemgrepInput() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestComponentsToComponentsDTO(t *testing.T) {
	tests := []struct {
		name    string
		request *common.ComponentsRequest
		want    []dtos.ComponentDTO
		wantErr bool
	}{
		{
			name: "Valid request with single component",
			request: &common.ComponentsRequest{
				Components: []*common.ComponentRequest{
					{
						Purl:        "pkg:npm/lodash",
						Requirement: "^4.17.0",
					},
				},
			},
			want: []dtos.ComponentDTO{
				{
					Purl:        "pkg:npm/lodash",
					Requirement: "^4.17.0",
				},
			},
			wantErr: false,
		},
		{
			name: "Valid request with multiple components",
			request: &common.ComponentsRequest{
				Components: []*common.ComponentRequest{
					{
						Purl:        "pkg:npm/lodash",
						Requirement: "^4.17.0",
					},
					{
						Purl:        "pkg:maven/org.apache/commons-lang",
						Requirement: ">=3.0.0",
					},
				},
			},
			want: []dtos.ComponentDTO{
				{
					Purl:        "pkg:npm/lodash",
					Requirement: "^4.17.0",
				},
				{
					Purl:        "pkg:maven/org.apache/commons-lang",
					Requirement: ">=3.0.0",
				},
			},
			wantErr: false,
		},
		{
			name:    "Nil request",
			request: nil,
			want:    []dtos.ComponentDTO{},
			wantErr: true,
		},
		{
			name: "Request with nil components",
			request: &common.ComponentsRequest{
				Components: nil,
			},
			want:    []dtos.ComponentDTO{},
			wantErr: true,
		},
		{
			name: "Request with empty components",
			request: &common.ComponentsRequest{
				Components: []*common.ComponentRequest{},
			},
			want:    []dtos.ComponentDTO{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := componentsToComponentsDTO(tt.request)

			if (err != nil) != tt.wantErr {
				t.Errorf("componentsToComponentsDTO() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("componentsToComponentsDTO() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConvertSemgrepResponse(t *testing.T) {
	tests := []struct {
		name    string
		output  dtos.SemgrepOutput
		wantErr bool
	}{
		{
			name: "Valid semgrep output with single purl",
			output: dtos.SemgrepOutput{
				Purls: []dtos.SemgrepOutputItem{
					{
						Purl:    "pkg:npm/lodash",
						Version: "4.17.21",
						Files: []dtos.SemgrepFileIssues{
							{
								File: "abc123",
								Path: "/src/index.js",
								Issues: []dtos.IssueItem{
									{
										RuleID:   "security.js.rule1",
										From:     "10",
										To:       "15",
										Severity: "high",
									},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Valid semgrep output with multiple purls",
			output: dtos.SemgrepOutput{
				Purls: []dtos.SemgrepOutputItem{
					{
						Purl:    "pkg:npm/lodash",
						Version: "4.17.21",
						Files: []dtos.SemgrepFileIssues{
							{
								File: "abc123",
								Path: "/src/index.js",
								Issues: []dtos.IssueItem{
									{
										RuleID:   "security.js.rule1",
										From:     "10",
										To:       "15",
										Severity: "high",
									},
								},
							},
						},
					},
					{
						Purl:    "pkg:npm/react",
						Version: "18.0.0",
						Files:   []dtos.SemgrepFileIssues{},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Empty semgrep output",
			output: dtos.SemgrepOutput{
				Purls: []dtos.SemgrepOutputItem{},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := convertSemgrepResponse(tt.output)

			if (err != nil) != tt.wantErr {
				t.Errorf("convertSemgrepResponse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && got == nil {
				t.Errorf("convertSemgrepResponse() returned nil response")
				return
			}

			if !tt.wantErr {
				if got.Purls == nil {
					t.Errorf("convertSemgrepResponse() returned response with nil Purls")
				}
			}
		})
	}
}
