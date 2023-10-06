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

// Package service implements the gRPC service endpoints
package service

import (
	"context"
	"errors"

	"github.com/jmoiron/sqlx"
	common "github.com/scanoss/papi/api/commonv2"
	pb "github.com/scanoss/papi/api/semgrepv2"
	myconfig "scanoss.com/semgrep/pkg/config"
	zlog "scanoss.com/semgrep/pkg/logger"
	"scanoss.com/semgrep/pkg/usecase"
)

type SemgrepServer struct {
	pb.SemgrepServer
	db     *sqlx.DB
	config *myconfig.ServerConfig
}

// NewSemgrepServer creates a new instance of Semgrep Server
func NewSemgrepServer(db *sqlx.DB, config *myconfig.ServerConfig) pb.SemgrepServer {
	return &SemgrepServer{db: db, config: config}
}

// Echo sends back the same message received
func (c SemgrepServer) Echo(ctx context.Context, request *common.EchoRequest) (*common.EchoResponse, error) {
	zlog.S.Infof("Received (%v): %v", ctx, request.GetMessage())
	return &common.EchoResponse{Message: request.GetMessage()}, nil
}

func (c SemgrepServer) GetIssues(ctx context.Context, request *common.PurlRequest) (*pb.SemgrepResponse, error) {

	//zlog.S.Infof("Processing Semgrep request: %v", request)
	// Make sure we have Semgrep data to query
	reqPurls := request.GetPurls()
	if reqPurls == nil || len(reqPurls) == 0 {
		statusResp := common.StatusResponse{Status: common.StatusCode_FAILED, Message: "No purls in request data supplied"}
		return &pb.SemgrepResponse{Status: &statusResp}, errors.New("no purl data supplied")
	}
	dtoRequest, err := convertSemgrepInput(request) // Convert to internal DTO for processing
	if err != nil {
		statusResp := common.StatusResponse{Status: common.StatusCode_FAILED, Message: "Problem parsing Semgrep input data"}
		return &pb.SemgrepResponse{Status: &statusResp}, errors.New("problem parsing Semgrep input data")
	}
	conn, err := c.db.Connx(ctx) // Get a connection from the pool
	if err != nil {
		zlog.S.Errorf("Failed to get a database connection from the pool: %v", err)
		statusResp := common.StatusResponse{Status: common.StatusCode_FAILED, Message: "Failed to get database pool connection"}
		return &pb.SemgrepResponse{Status: &statusResp}, errors.New("problem getting database pool connection")
	}
	defer closeDbConnection(conn)
	// Search the KB for information about each Semgrepgraphy
	semgrepUc := usecase.NewSemgrep(ctx, conn)
	dtoSemgrep, err := semgrepUc.GetIssues(dtoRequest)

	if err != nil {
		zlog.S.Errorf("Failed to get Issues: %v", err)
		statusResp := common.StatusResponse{Status: common.StatusCode_FAILED, Message: "Problems encountered extracting Semgrep data"}
		return &pb.SemgrepResponse{Status: &statusResp}, nil
	}
	//zlog.S.Debugf("Parsed Semgrep: %+v", dtoSemgrep)

	semgrepResponse, err := convertSemgrepOutput(dtoSemgrep) // Convert the internal data into a response object
	if err != nil {
		zlog.S.Errorf("Failed to covnert parsed dependencies: %v", err)
		statusResp := common.StatusResponse{Status: common.StatusCode_FAILED, Message: "Problems encountered extracting Semgrep data"}
		return &pb.SemgrepResponse{Status: &statusResp}, nil
	}
	// Set the status and respond with the data
	statusResp := common.StatusResponse{Status: common.StatusCode_SUCCESS, Message: "Success"}
	return &pb.SemgrepResponse{Purls: semgrepResponse.Purls, Status: &statusResp}, nil
}

// closeDbConnection closes the specified database connection
func closeDbConnection(conn *sqlx.Conn) {
	zlog.S.Debugf("Closing DB Connection: %v", conn)
	err := conn.Close()
	if err != nil {
		zlog.S.Warnf("Warning: Problem closing database connection: %v", err)
	}
}
