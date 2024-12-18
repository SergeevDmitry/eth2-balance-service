// Copyright 2021 stafiprotocol
// SPDX-License-Identifier: LGPL-3.0-only

package cmd

import (
	"fmt"

	"github.com/SergeevDmitry/eth2-balance-service/dao"
	"github.com/SergeevDmitry/eth2-balance-service/pkg/config"
	"github.com/SergeevDmitry/eth2-balance-service/pkg/db"
	"github.com/SergeevDmitry/eth2-balance-service/pkg/log"
	task_v1_syncer "github.com/SergeevDmitry/eth2-balance-service/task/v1_syncer"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func startV1SyncerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync-v1-data",
		Short: "Sync data of v1",
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, err := cmd.Flags().GetString(flagConfigPath)
			if err != nil {
				return err
			}
			fmt.Printf("config path: %s\n", configPath)

			logLevelStr, err := cmd.Flags().GetString(flagLogLevel)
			if err != nil {
				return err
			}
			logLevel, err := logrus.ParseLevel(logLevelStr)
			if err != nil {
				return err
			}
			logrus.SetLevel(logLevel)

			cfg, err := config.Load(configPath)
			if err != nil {
				return err
			}
			logrus.Infof(
				`v1 syncer config info:
	logFilePath: %s
	logLevel: %s
	eth1Endpoint: %s
	eth2Endpoint: %s`,
				cfg.LogFilePath, logLevelStr, cfg.Eth1Endpoint, cfg.Eth2Endpoint)

			err = log.InitLogFile(cfg.LogFilePath + "/v1_syncer")
			if err != nil {
				return err
			}
			//init db
			db, err := db.NewDB(&db.Config{
				Host:     cfg.Db.Host,
				Port:     cfg.Db.Port,
				User:     cfg.Db.User,
				Pass:     cfg.Db.Pwd,
				DBName:   cfg.Db.Name,
				LogLevel: logLevelStr})
			if err != nil {
				logrus.Errorf("db err: %s", err)
				return err
			}
			logrus.Infof("db connect success")

			//interrupt signal
			defer func() {
				sqlDb, err := db.DB.DB()
				if err != nil {
					logrus.Errorf("db.DB() err: %s", err)
					return
				}
				logrus.Infof("shutting down the db ...")
				sqlDb.Close()
			}()
			err = dao.AutoMigrate(db)
			if err != nil {
				logrus.Errorf("dao autoMigrate err: %s", err)
				return err
			}

			t, err := task_v1_syncer.NewTask(cfg, db)
			if err != nil {
				return err
			}
			err = t.Start()
			if err != nil {
				logrus.Errorf("task start err: %s", err)
				return err
			}
			defer func() {
				logrus.Infof("shutting down task ...")
				t.Stop()
			}()

			return nil
		},
	}

	cmd.Flags().String(flagConfigPath, defaultConfigPath, "Config file path")
	cmd.Flags().String(flagLogLevel, logrus.InfoLevel.String(), "The logging level (trace|debug|info|warn|error|fatal|panic)")

	return cmd
}
