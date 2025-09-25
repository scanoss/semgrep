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

package cmd

import (
	"context"
	_ "embed"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/golobby/config/v3"
	"github.com/golobby/config/v3/pkg/feeder"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"go.uber.org/zap/zapcore"

	myconfig "scanoss.com/semgrep/pkg/config"
	zlog "scanoss.com/semgrep/pkg/logger"
	m "scanoss.com/semgrep/pkg/models"
	"scanoss.com/semgrep/pkg/protocol/grpc"
	"scanoss.com/semgrep/pkg/service"
)

//go:generate bash ../../get_version.sh
//go:embed version.txt
var version string

// getConfig checks command line args for option to feed into the config parser.
func getConfig() (*myconfig.ServerConfig, error) {
	var jsonConfig, envConfig string
	flag.StringVar(&jsonConfig, "json-config", "", "Application JSON config")
	flag.StringVar(&envConfig, "env-config", "", "Application dot-ENV config")
	debug := flag.Bool("debug", false, "Enable debug")
	ver := flag.Bool("version", false, "Display current version")
	flag.Parse()
	if *ver {
		fmt.Printf("Version: %v", version)
		os.Exit(1)
	}
	var feeders []config.Feeder
	if len(jsonConfig) > 0 {
		feeders = append(feeders, feeder.Json{Path: jsonConfig})
	}
	if len(envConfig) > 0 {
		feeders = append(feeders, feeder.DotEnv{Path: envConfig})
	}
	if *debug {
		err := os.Setenv("APP_DEBUG", "1")
		if err != nil {
			fmt.Printf("Warning: Failed to set env APP_DEBUG to 1: %v", err)
			return nil, err
		}
	}
	myConfig, err := myconfig.NewServerConfig(feeders)
	return myConfig, err
}

// closeDBConnection closes the specified DB connection.
func closeDBConnection(db *sqlx.DB) {
	err := db.Close()
	if err != nil {
		zlog.S.Warnf("Problem closing DB: %v", err)
	}
}

// RunServer runs the gRPC semgrep Server.
func RunServer() error {
	// Load command line options and config
	cfg, err := getConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %v", err)
	}
	// Check mode to determine which logger to load
	switch strings.ToLower(cfg.App.Mode) {
	case "prod":
		logErr := error(nil)
		if cfg.App.Debug {
			logErr = zlog.NewSugaredProdLoggerLevel(zapcore.DebugLevel)
		} else {
			logErr = zlog.NewSugaredProdLogger()
		}
		if logErr != nil {
			return fmt.Errorf("failed to load logger: %v", logErr)
		}
		zlog.L.Debug("Running with debug enabled")
	default:
		if logErr := zlog.NewSugaredDevLogger(); logErr != nil {
			return fmt.Errorf("failed to load logger: %v", logErr)
		}
	}
	defer zlog.SyncZap()
	zlog.S.Infof("Starting SCANOSS semgrep Service: %v", strings.TrimSpace(version))
	// Setup database connection pool
	var dsn string
	if len(cfg.Database.Dsn) > 0 {
		dsn = cfg.Database.Dsn
	} else {
		dsn = fmt.Sprintf("%s://%s:%s@%s/%s?sslmode=%s",
			cfg.Database.Driver,
			cfg.Database.User,
			cfg.Database.Passwd,
			cfg.Database.Host,
			cfg.Database.Schema,
			cfg.Database.SslMode)
	}
	zlog.S.Debug("Connecting to Database...")
	db, err := sqlx.Open(cfg.Database.Driver, dsn)
	if err != nil {
		zlog.S.Errorf("Failed to open database: %v", err)
		return fmt.Errorf("failed to open database: %v", err)
	}
	db.SetConnMaxIdleTime(30 * time.Minute) // TODO add to app config
	db.SetConnMaxLifetime(time.Hour)
	db.SetMaxIdleConns(20)
	db.SetMaxOpenConns(100)
	err = db.Ping()
	if err != nil {
		zlog.S.Errorf("Failed to ping database: %v", err)
		return fmt.Errorf("failed to ping database: %v", err)
	}
	m.LDBBinPath = cfg.LDB.BinPath
	m.LDBEncBinPath = cfg.LDB.EncBinPath

	m.LDBPivotTableName = cfg.LDB.PivotName
	m.LDBFileTableName = cfg.LDB.FileName
	m.LDBSemgreptTableName = cfg.LDB.SemgrepName

	tables, errLDB := m.PingLDB("oss")
	if errLDB != nil {
		zlog.S.Errorf("Failed to ping LDB: %v", errLDB)
		return fmt.Errorf("failed to ping LDB: %v", errLDB)
	}
	if !m.ContainsTable(tables, cfg.LDB.SemgrepName) {
		zlog.S.Error("semgrep LDB table not found")
		//	return fmt.Errorf("%s", "semgrep LDB table not found")
	}
	if !m.ContainsTable(tables, cfg.LDB.FileName) {
		fmt.Println(cfg.LDB.FileName)
		zlog.S.Error("File LDB table not found")
		return fmt.Errorf("%s", "file LDB table not found")
	}
	if !m.ContainsTable(tables, cfg.LDB.PivotName) {
		zlog.S.Error("Pivot LDB table not found")
		return fmt.Errorf("%s", "Pivot LDB table not found")
	}

	defer closeDBConnection(db)
	v2API := service.NewSemgrepServer(db, cfg)
	ctx := context.Background()
	return grpc.RunServer(ctx, v2API, cfg.App.Port)
}
