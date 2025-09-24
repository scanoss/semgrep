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

package service

import (
	"encoding/json"

	common "github.com/scanoss/papi/api/commonv2"
	pb "github.com/scanoss/papi/api/semgrepv2"
	"scanoss.com/semgrep/pkg/dtos"
	se "scanoss.com/semgrep/pkg/errors"
	zlog "scanoss.com/semgrep/pkg/logger"
)

// convertPurlRequestInput converts a Purl Request structure into an internal Semgrep Input struct
func convertSemgrepInput(request *common.PurlRequest) ([]dtos.ComponentDTO, error) {
	if request == nil || request.Purls == nil || len(request.Purls) == 0 {
		return []dtos.ComponentDTO{}, se.NewBadRequestError("Request validation failed: 'purls' array is required and must contain at least one component", nil)
	}
	componentDTOS := make([]dtos.ComponentDTO, 0, len(request.Purls))
	for i, purl := range request.Purls {
		componentDTOS[i] = dtos.ComponentDTO{
			Purl:        purl.Purl,
			Requirement: purl.Requirement,
		}
	}
	return componentDTOS, nil
}

// convertSemgrepOutput converts an internal Semgrep Output structure into a SemgrepResponse struct
func convertSemgrepResponse(output dtos.SemgrepOutput) (*pb.SemgrepResponse, error) {
	data, err := json.Marshal(output)
	if err != nil {
		zlog.S.Errorf("Problem marshalling Semgrep request output: %v", err)
		return &pb.SemgrepResponse{}, se.NewInternalError("Problem marshalling Semgrep request output", err)
	}
	//zlog.S.Debugf("Parsed data: %v", string(data))
	var depResp pb.SemgrepResponse
	err = json.Unmarshal(data, &depResp)
	if err != nil {
		zlog.S.Errorf("Problem unmarshalling Semgrep request output: %v", err)
		return &pb.SemgrepResponse{}, se.NewInternalError("Problem unmarshalling Semgrep request output", err)
	}
	return &depResp, nil
}
