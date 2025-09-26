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

// This file common tasks for the models package

package models

import (
	"context"
	"fmt"
	"io/ioutil"

	"github.com/jmoiron/sqlx"
	zlog "github.com/scanoss/zap-logging-helper/pkg/logger"
)

// loadSQLData Load the specified SQL files into the supplied DB.
func loadSQLData(db *sqlx.DB, ctx context.Context, filename string) error {
	fmt.Printf("Loading test data file: %v\n", filename)
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, string(file))
	if err != nil {
		return err
	}
	return nil
}

// LoadTestSQLData loads all the required test SQL files.
func LoadTestSQLData(db *sqlx.DB, ctx context.Context) error {
	files := []string{"./tests/mines.sql", "./tests/all_urls.sql", "./tests/projects.sql",
		"./tests/licenses.sql", "./tests/versions.sql", "./tests/golang_projects.sql"}
	return loadTestSQLDataFiles(db, ctx, files)
}

// loadTestSQLDataFiles loads a list of test SQL files.
func loadTestSQLDataFiles(db *sqlx.DB, ctx context.Context, files []string) error {
	for _, file := range files {
		err := loadSQLData(db, ctx, file)
		if err != nil {
			return err
		}
	}
	return nil
}

// CloseDB closes the specified DB and logs any errors.
func CloseDB(db *sqlx.DB) {
	if db != nil {
		zlog.S.Debugf("Closing DB...")
		err := db.Close()
		if err != nil {
			zlog.S.Warnf("Problem closing DB: %v", err)
		}
	}
}

// CloseConn closes the specified DB connection and logs any errors.
func CloseConn(conn *sqlx.Conn) {
	if conn != nil {
		zlog.S.Debugf("Closing Connection...")
		err := conn.Close()
		if err != nil {
			zlog.S.Warnf("Problem closing DB connection: %v", err)
		}
	}
}

// CloseRows closes the specified DB query row and logs any errors.
func CloseRows(rows *sqlx.Rows) {
	if rows != nil {
		zlog.S.Debugf("Closing Rows...")
		err := rows.Close()
		if err != nil {
			zlog.S.Warnf("Problem closing Rows: %v", err)
		}
	}
}
