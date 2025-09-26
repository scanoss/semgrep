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

// Package service provides Semgrep support utilities and conversion functions.
//
// This package contains helper functions for converting between different data structures
// used in the Semgrep service, including:
//   - Converting protobuf request structures to internal DTOs
//   - Converting internal DTOs to protobuf response structures
package service

import (
	"encoding/json"

	common "github.com/scanoss/papi/api/commonv2"
	pb "github.com/scanoss/papi/api/semgrepv2"

	zlog "github.com/scanoss/zap-logging-helper/pkg/logger"
	"scanoss.com/semgrep/pkg/dtos"
	se "scanoss.com/semgrep/pkg/errors"
)

// convertSemgrepInput converts a PurlRequest protobuf structure to a slice of ComponentDTO.
// Parameters:
//   - request: A pointer to common.PurlRequest containing PURL data to convert
//
// Returns:
//   - []dtos.ComponentDTO: A slice of converted component DTOs
//   - error: BadRequestError if the request is nil, empty, or invalid
func convertSemgrepInput(request *common.PurlRequest) ([]dtos.ComponentDTO, error) {
	if request == nil || request.Purls == nil || len(request.Purls) == 0 {
		return []dtos.ComponentDTO{}, se.NewBadRequestError("Request validation failed: 'purls' array is required and must contain at least one component", nil)
	}
	componentDTOS := make([]dtos.ComponentDTO, 0, len(request.Purls))
	for _, purl := range request.Purls {
		componentDTOS = append(componentDTOS, dtos.ComponentDTO{
			Purl:        purl.Purl,
			Requirement: purl.Requirement,
		})
	}
	return componentDTOS, nil
}

// componentsToComponentsDTO converts a ComponentsRequest protobuf structure to a slice of ComponentDTO.
// Parameters:
//   - request: A pointer to common.ComponentsRequest containing component data to convert
//
// Returns:
//   - []dtos.ComponentDTO: A slice of converted component DTOs
//   - error: BadRequestError if the request is nil, empty, or invalid
func componentsToComponentsDTO(request *common.ComponentsRequest) ([]dtos.ComponentDTO, error) {
	if request == nil || request.Components == nil || len(request.Components) == 0 {
		return []dtos.ComponentDTO{}, se.NewBadRequestError("Request validation failed: 'purls' array is required and must contain at least one component", nil)
	}
	componentDTOS := make([]dtos.ComponentDTO, 0, len(request.Components))
	for _, purl := range request.Components {
		componentDTOS = append(componentDTOS, dtos.ComponentDTO{
			Purl:        purl.Purl,
			Requirement: purl.Requirement,
		})
	}
	return componentDTOS, nil
}

// convertSemgrepResponse converts internal SemgrepOutput to a SemgrepResponse protobuf structure.
// Parameters:
//   - output: dtos.SemgrepOutput containing the internal Semgrep analysis results
//
// Returns:
//   - *pb.SemgrepResponse: A pointer to the converted protobuf response structure
//   - error: InternalError if JSON marshaling/unmarshaling fails
func convertSemgrepResponse(output dtos.SemgrepOutput) (*pb.SemgrepResponse, error) {
	data, err := json.Marshal(output)
	if err != nil {
		zlog.S.Errorf("Problem marshalling Semgrep request output: %v", err)
		return &pb.SemgrepResponse{}, se.NewInternalError("Problem marshalling Semgrep request output", err)
	}
	// zlog.S.Debugf("Parsed data: %v", string(data))
	var depResp pb.SemgrepResponse
	err = json.Unmarshal(data, &depResp)
	if err != nil {
		zlog.S.Errorf("Problem unmarshalling Semgrep request output: %v", err)
		return &pb.SemgrepResponse{}, se.NewInternalError("Problem unmarshalling Semgrep request output", err)
	}
	return &depResp, nil
}

// convertToComponentsIssues converts internal SemgrepOutput to ComponentsIssueResponse protobuf structure.
// Parameters:
//   - output: dtos.SemgrepOutput containing the internal Semgrep analysis results
//
// Returns:
//   - *pb.ComponentsIssueResponse: A pointer to the response containing issues for all components
//   - error: InternalError if JSON marshaling/unmarshaling fails during file processing
func convertToComponentsIssues(output dtos.SemgrepOutput) (*pb.ComponentsIssueResponse, error) {
	response := &pb.ComponentsIssueResponse{
		Components: []*pb.ComponentIssueInfo{},
	}
	for _, o := range output.Purls {
		files := []*pb.File{}
		data, err := json.Marshal(o.Files)
		if err != nil {
			return &pb.ComponentsIssueResponse{}, se.NewInternalError("Problem marshalling Semgrep request output", err)
		}
		err = json.Unmarshal(data, &files)
		if err != nil {
			return &pb.ComponentsIssueResponse{}, se.NewInternalError("Problem unmarshalling Semgrep request output", err)
		}
		componentIssueInfo := &pb.ComponentIssueInfo{
			Purl:        o.Purl,
			Version:     o.Version,
			Requirement: o.Version,
			Files:       files,
		}
		response.Components = append(response.Components, componentIssueInfo)
	}
	return response, nil
}

// convertToComponentIssues converts internal SemgrepOutput to a single ComponentIssueResponse.
//
// Parameters:
//   - output: dtos.SemgrepOutput containing the internal Semgrep analysis results
//
// Returns:
//   - *pb.ComponentIssueResponse: A pointer to the response containing issues for the first component
//   - error: InternalError if conversion fails, NotFoundError if no components found
func convertToComponentIssues(output dtos.SemgrepOutput) (*pb.ComponentIssueResponse, error) {
	componentsIssuesResponse, err := convertToComponentsIssues(output)
	if err != nil {
		return &pb.ComponentIssueResponse{}, err
	}
	if componentsIssuesResponse.Components == nil && len(componentsIssuesResponse.Components) == 0 {
		return &pb.ComponentIssueResponse{}, se.NewNotFoundError("Component not found")
	}
	return &pb.ComponentIssueResponse{Component: componentsIssuesResponse.Components[0]}, nil
}
