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

	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
	"scanoss.com/semgrep/pkg/dtos"
	se "scanoss.com/semgrep/pkg/errors"

	"github.com/jmoiron/sqlx"
	common "github.com/scanoss/papi/api/commonv2"
	pb "github.com/scanoss/papi/api/semgrepv2"
	zlog "github.com/scanoss/zap-logging-helper/pkg/logger"
	myconfig "scanoss.com/semgrep/pkg/config"
	"scanoss.com/semgrep/pkg/usecase"
)

// SemgrepServer implements the gRPC service for Semgrep operations.
// It handles security vulnerability scanning requests and manages the interaction
// between the gRPC layer and the business logic layer.
type SemgrepServer struct {
	pb.SemgrepServer
	db             *sqlx.DB                // Database connection for persistence
	config         *myconfig.ServerConfig  // Server configuration settings
	semgrepUseCase *usecase.SemgrepUseCase // Business logic handler for Semgrep operations
}

// NewSemgrepServer creates a new instance of Semgrep Server.
// It initializes the server with database connection, configuration, and use case handlers.
//
// Parameters:
//   - db: Database connection for data operations
//   - config: Server configuration settings
//
// Returns:
//   - pb.SemgrepServer: Initialized gRPC server instance
func NewSemgrepServer(db *sqlx.DB, config *myconfig.ServerConfig) pb.SemgrepServer {
	return &SemgrepServer{
		db:             db,
		config:         config,
		semgrepUseCase: usecase.NewSemgrep(db),
	}
}

// Echo sends back the same message received.
// This endpoint is used for health checks and connectivity testing.
//
// Parameters:
//   - ctx: Request context for cancellation and timeout
//   - request: Echo request containing the message to echo back
//
// Returns:
//   - *common.EchoResponse: Response containing the echoed message
//   - error: Always nil as this endpoint doesn't fail
func (c SemgrepServer) Echo(ctx context.Context, request *common.EchoRequest) (*common.EchoResponse, error) {
	zlog.S.Infof("Received (%v): %v", ctx, request.GetMessage())
	return &common.EchoResponse{Message: request.GetMessage()}, nil
}

// ResponseBuilder is a generic function type for building typed responses from Semgrep output.
// It takes the processing result and any errors, then constructs the appropriate response type.
type ResponseBuilder[T any] func(ctx context.Context, s *zap.SugaredLogger, semgrep dtos.SemgrepOutput, err error) T

// RequestConverter is a generic function type for converting incoming gRPC requests
// to internal ComponentDTO format used by the business logic layer.
type RequestConverter[R any] func(R) ([]dtos.ComponentDTO, error)

// UseCaseHandler defines the function signature for business logic handlers.
// It processes component DTOs and returns Semgrep analysis results.
type UseCaseHandler func(context.Context, *zap.SugaredLogger, []dtos.ComponentDTO) (dtos.SemgrepOutput, error)

// handleLegacyRequest provides a generic request handling pattern for legacy endpoints.
// It orchestrates the request conversion, business logic execution, and response building.
//
// Type Parameters:
//   - R: Request type
//   - T: Response type
//
// Parameters:
//   - ctx: Request context
//   - req: The incoming request
//   - useCaseHandler: Function to execute business logic
//   - requestConverter: Function to convert request to internal format
//   - responseBuilder: Function to build the typed response
//
// Returns:
//   - T: The built response of the specified type
func handleLegacyRequest[R any, T any](
	ctx context.Context,
	req R,
	useCaseHandler UseCaseHandler,
	requestConverter RequestConverter[R],
	responseBuilder ResponseBuilder[T],
) T {
	s := ctxzap.Extract(ctx).Sugar()
	dtoRequest, err := requestConverter(req) // Convert to internal DTO for processing
	if err != nil {
		responseBuilder(ctx, s, dtos.SemgrepOutput{}, err)
	}

	dtoSemgrep, err := useCaseHandler(ctx, s, dtoRequest)
	return responseBuilder(ctx, s, dtoSemgrep, err)
}

// 3. Build response with appropriate status codes.
func (c SemgrepServer) GetIssues(ctx context.Context, request *common.PurlRequest) (*pb.SemgrepResponse, error) {
	response := handleLegacyRequest(
		ctx,
		request,
		c.semgrepUseCase.GetIssues,
		convertSemgrepInput,
		// Response builder for SemgrepResponse
		func(ctx context.Context, s *zap.SugaredLogger, semgrep dtos.SemgrepOutput, err error) *pb.SemgrepResponse {
			statusResp := &common.StatusResponse{Status: common.StatusCode_SUCCESS, Message: "Success"}
			if err != nil {
				statusResp = se.HandleServiceError(ctx, s, err)
				return &pb.SemgrepResponse{Status: statusResp}
			}
			resp, err := convertSemgrepResponse(semgrep)
			if err != nil {
				statusResp = se.HandleServiceError(ctx, s, err)
				return &pb.SemgrepResponse{Status: statusResp}
			}
			resp.Status = statusResp
			return resp
		})
	return response, nil
}

func (c SemgrepServer) GetComponentsIssues(ctx context.Context, request *common.ComponentsRequest) (*pb.ComponentsIssueResponse, error) {
	response := handleLegacyRequest(
		ctx,
		request,
		c.semgrepUseCase.GetIssues,
		componentsToComponentsDTO,
		// Response builder for SemgrepResponse
		func(ctx context.Context, s *zap.SugaredLogger, semgrep dtos.SemgrepOutput, err error) *pb.ComponentsIssueResponse {
			statusResp := &common.StatusResponse{Status: common.StatusCode_SUCCESS, Message: "Success"}
			if err != nil {
				statusResp = se.HandleServiceError(ctx, s, err)
				return &pb.ComponentsIssueResponse{Status: statusResp}
			}
			resp, err := convertToComponentsIssues(semgrep)
			if err != nil {
				statusResp = se.HandleServiceError(ctx, s, err)
				return &pb.ComponentsIssueResponse{Status: statusResp}
			}
			resp.Status = statusResp
			return resp
		})
	return response, nil
}

func (c SemgrepServer) GetComponentIssues(ctx context.Context, request *common.ComponentRequest) (*pb.ComponentIssueResponse, error) {
	req := &common.ComponentsRequest{
		Components: []*common.ComponentRequest{request},
	}
	response := handleLegacyRequest(
		ctx,
		req,
		c.semgrepUseCase.GetIssues,
		componentsToComponentsDTO,
		// Response builder for SemgrepResponse
		func(ctx context.Context, s *zap.SugaredLogger, semgrep dtos.SemgrepOutput, err error) *pb.ComponentIssueResponse {
			statusResp := &common.StatusResponse{Status: common.StatusCode_SUCCESS, Message: "Success"}
			if err != nil {
				statusResp = se.HandleServiceError(ctx, s, err)
				return &pb.ComponentIssueResponse{Status: statusResp}
			}
			resp, err := convertToComponentIssues(semgrep)
			if err != nil {
				statusResp = se.HandleServiceError(ctx, s, err)
				return &pb.ComponentIssueResponse{Status: statusResp}
			}
			resp.Status = statusResp
			return resp
		})
	return response, nil
}
