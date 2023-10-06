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
	"errors"

	common "github.com/scanoss/papi/api/commonv2"
	pb "github.com/scanoss/papi/api/semgrepv2"
	"scanoss.com/semgrep/pkg/dtos"
	zlog "scanoss.com/semgrep/pkg/logger"
)

// convertPurlRequestInput converts a Purl Request structure into an internal Semgrep Input struct
func convertSemgrepInput(request *common.PurlRequest) (dtos.SemgrepInput, error) {
	data, err := json.Marshal(request)
	if err != nil {
		zlog.S.Errorf("Problem marshalling Semgrep request input: %v", err)
		return dtos.SemgrepInput{}, errors.New("problem marshalling Semgrep input")
	}
	dtoRequest, err := dtos.ParseSemgrepInput(data)
	if err != nil {
		zlog.S.Errorf("Problem parsing Semgrep request input: %v", err)
		return dtos.SemgrepInput{}, errors.New("problem parsing Semgrep input")
	}
	return dtoRequest, nil
}

// convertSemgrepOutput converts an internal Semgrep Output structure into a SemgrepResponse struct
func convertSemgrepOutput(output dtos.SemgrepOutput) (*pb.SemgrepResponse, error) {
	data, err := json.Marshal(output)
	if err != nil {
		zlog.S.Errorf("Problem marshalling Semgrep request output: %v", err)
		return &pb.SemgrepResponse{}, errors.New("problem marshalling Semgrep output")
	}
	//zlog.S.Debugf("Parsed data: %v", string(data))
	var depResp pb.SemgrepResponse
	err = json.Unmarshal(data, &depResp)
	if err != nil {
		zlog.S.Errorf("Problem unmarshalling Semgrep request output: %v", err)
		return &pb.SemgrepResponse{}, errors.New("problem unmarshalling Semgrep output")
	}
	return &depResp, nil
}
